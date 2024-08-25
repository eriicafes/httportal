package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/eriicafes/httportal/views/pages"
	"github.com/eriicafes/httportal/views/partials"
	"github.com/eriicafes/tmpl"
)

type App struct {
	tmpl.Templates
	portal *Portal
}

func New(tp tmpl.Templates, p *Portal) *App {
	return &App{Templates: tp, portal: p}
}

func (app *App) Mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", app.withError(app.home))
	mux.HandleFunc("GET /send", app.withError(app.send))
	mux.HandleFunc("POST /send", app.withError(app.sendPost))
	mux.HandleFunc("GET /receive", app.withError(app.receive))
	mux.HandleFunc("POST /receive", app.withError(app.receivePost))
	mux.Handle("POST /transfer/{id}", http.TimeoutHandler(app.withError(app.transferUpload), time.Minute*5, "Upload timed out"))
	mux.Handle("GET /transfer/{id}", http.TimeoutHandler(app.withError(app.transferDownload), time.Minute*5, "Download timed out"))
	mux.HandleFunc("GET /transfer/{id}/events", app.withError(app.transferEvents))
}

func (app *App) home(w http.ResponseWriter, r *http.Request) error {
	return app.Render(w, pages.IndexPage{})
}

func (app *App) send(w http.ResponseWriter, r *http.Request) error {
	return app.Render(w, pages.SendPage{})
}

func (app *App) sendPost(w http.ResponseWriter, r *http.Request) error {
	// create connection
	id, err := app.portal.CreateConnection()
	if err != nil {
		return NewClientError(err, "Failed to create connection").
			WithStatus(http.StatusInternalServerError)
	}
	// set peer cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Session",
		Value:    PeerSender.Pid(id),
		Path:     fmt.Sprintf("/transfer/%s", id),
		Expires:  time.Now().Add(time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	if err = app.RenderAssociated(w, pages.SendForm{ID: id}); err != nil {
		return err
	}
	// send activity connector oob partial
	return app.RenderAssociated(w, partials.ActivityConnector{ID: id})
}

func (app *App) receive(w http.ResponseWriter, r *http.Request) error {
	id := r.URL.Query().Get("id")
	if id == "" {
		return app.Render(w, pages.ReceivePage{})
	}
	// get connection
	conn, err := app.portal.GetConnection(id)
	// check if connection is open
	if err == nil && conn.CanEnter(PeerReceiver) {
		// set peer cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "Session",
			Value:    PeerReceiver.Pid(id),
			Path:     fmt.Sprintf("/transfer/%s", id),
			Expires:  time.Now().Add(time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
	} else {
		id = ""
	}
	return app.Render(w, pages.ReceivePage{ID: id})
}

func (app *App) receivePost(w http.ResponseWriter, r *http.Request) error {
	id := r.FormValue("id")
	// get connection
	conn, err := app.portal.GetConnection(id)
	if err != nil {
		desc := "The connection is invalid or expired."
		if len(id) < idLen {
			desc = "Invalid connection ID."
		}
		if len(id) > idLen && strings.HasPrefix(id, "http") {
			desc = "Invalid connection ID, open the link to join connection."
		}
		return NewClientError(err, "Connection not found").
			WithDesc(desc).
			WithStatus(http.StatusNotFound)
	}
	// check if connection is open
	if !conn.CanEnter(PeerReceiver) {
		return NewClientError(nil, "Connection not available").
			WithDesc("Receiver already joined this connection.")
	}
	// set peer cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Session",
		Value:    PeerReceiver.Pid(id),
		Path:     fmt.Sprintf("/transfer/%s", id),
		Expires:  time.Now().Add(time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	if err = app.RenderAssociated(w, pages.ReceiveForm{ID: id}); err != nil {
		return err
	}
	// send activity connector oob partial
	return app.RenderAssociated(w, partials.ActivityConnector{ID: id})
}

func (app *App) transferUpload(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	// get peer from cookie
	cookie, err := r.Cookie("Session")
	if err != nil {
		return NewClientError(err, "Unauthorized to send").
			WithDesc("Create a new connection to send.").
			WithStatus(http.StatusUnauthorized)
	}
	peer, err := ParsePeer(id, cookie.Value)
	if err != nil {
		return NewClientError(err, "Unauthorized to send").
			WithDesc("Create a new connection to send.").
			WithStatus(http.StatusUnauthorized)
	}
	// verify peer is sender
	if peer != PeerSender {
		return NewClientError(err, "Unauthorized to send").
			WithDesc("Only sender is allowed to send.").
			WithStatus(http.StatusUnauthorized)
	}
	// get connection
	conn, err := app.portal.GetConnection(id)
	if err != nil {
		return NewClientError(err, "Connection not found").
			WithDesc("The connection has expired.").
			WithStatus(http.StatusNotFound)
	}
	// enter connection
	err = conn.Enter(peer)
	if err != nil {
		return NewClientError(err, "Connection not available").
			WithDesc("Sender already joined this connection.")
	}
	conn.Broadcast(Mssg{Data: "Sender has joined connection"})

	// start goroutine to close connection on request end
	go func(ctx context.Context, conn *Conn) {
		<-ctx.Done()
		conn.CloseWriter()
	}(r.Context(), conn)

	// handle upload
	file, header, err := r.FormFile("file")
	if err != nil {
		return NewClientError(err, "Upload failed").WithDesc("Failed to parse uploaded file.")
	}
	contentType, err := detectContentType(file)
	if err != nil {
		return NewClientError(err, "Upload failed").WithDesc("Failed to parse file type.")
	}
	conn.Broadcast(Mssg{Data: "Waiting to upload"})
	conn.SendHeaders(Headers{ContentType: contentType, FileHeader: header})
	conn.Broadcast(Mssg{Data: "Uploading..."})

	// start goroutine to broadcast upload progress every second
	go func(ctx context.Context, size int64) {
		var sum int64 = 0
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				percentage := fmt.Sprintf("%.0f%%", (float64(sum)/float64(size))*100)
				conn.Broadcast(Mssg{Event: "progress", Data: percentage})
			case progress := <-conn.Progress():
				sum += int64(progress)
			case <-ctx.Done():
				return
			}
		}
	}(r.Context(), header.Size)

	_, err = conn.Send(file)
	if err != nil {
		conn.Broadcast(Mssg{Data: "Upload failed"})
	} else {
		conn.Broadcast(Mssg{Event: "progress", Data: "100%"})
		conn.Broadcast(Mssg{Data: "Upload complete"})
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (app *App) transferDownload(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	// get peer from cookie
	cookie, err := r.Cookie("Session")
	if err != nil {
		return NewClientError(err, "Unauthorized to receive").
			WithDesc("Join a connection to receive.").
			WithStatus(http.StatusUnauthorized)
	}
	peer, err := ParsePeer(id, cookie.Value)
	if err != nil {
		return NewClientError(err, "Unauthorized to receive").
			WithDesc("Join a connection to receive.").
			WithStatus(http.StatusUnauthorized)
	}
	// verify peer is receiver
	if peer != PeerReceiver {
		return NewClientError(err, "Unauthorized to receive").
			WithDesc("Only receiver is allowed to receive.").
			WithStatus(http.StatusUnauthorized)
	}
	// get connection
	conn, err := app.portal.GetConnection(id)
	if err != nil {
		return NewClientError(err, "Connection not found").
			WithDesc("The connection has expired.").
			WithStatus(http.StatusNotFound)
	}
	// enter connection
	err = conn.Enter(peer)
	if err != nil {
		return NewClientError(err, "Connection not available").
			WithDesc("Receiver already joined this connection.")
	}
	conn.Broadcast(Mssg{Data: "Receiver has joined connection"})

	// start goroutine to close connection on request end
	go func(ctx context.Context, conn *Conn) {
		<-ctx.Done()
		conn.CloseReader()
	}(r.Context(), conn)

	// handle download
	conn.Broadcast(Mssg{Data: "Waiting to download"})
	headers := conn.ReceiveHeaders()
	w.Header().Add("Content-Type", headers.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", headers.Filename))
	w.Header().Add("Content-Length", fmt.Sprint(headers.Size))
	conn.Broadcast(Mssg{Data: "Downloading..."})

	_, err = conn.Receive(w)
	if err != nil {
		conn.Broadcast(Mssg{Data: "Download failed"})
	} else {
		conn.Broadcast(Mssg{Data: "Download complete"})
	}
	return nil
}

func (app *App) transferEvents(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	rc := http.NewResponseController(w)
	rc.Flush()

	id := r.PathValue("id")
	// get peer from cookie
	cookie, err := r.Cookie("Session")

	// close connection on error to prevent client from reconnecting
	if err != nil {
		fmt.Fprint(w, Mssg{Event: "close", Data: "Unauthorized"})
		return nil
	}
	peer, err := ParsePeer(id, cookie.Value)
	if err != nil {
		fmt.Fprint(w, Mssg{Event: "close", Data: "Unauthorized"})
		return nil
	}
	// get connection
	conn, err := app.portal.GetConnection(id)
	if err != nil {
		fmt.Fprint(w, Mssg{Event: "close", Data: "Not Found"})
		return nil
	}

	ping := time.NewTicker(time.Second)
	defer ping.Stop()
	for {
		select {
		case mssg, ok := <-conn.Mssg(peer):
			if !ok {
				fmt.Fprint(w, Mssg{Event: "close", Data: "Done"})
				return nil
			}
			mssg.Data = app.mssgToHTML(mssg)
			fmt.Fprint(w, mssg)
			rc.Flush()
		case <-r.Context().Done():
			return nil
		case <-ping.C:
			// keep select running every second to skip blocking cases
			// a default case will do the same thing but too often
			continue
		}
	}
}

func (app *App) mssgToHTML(mssg Mssg) string {
	var err error
	var html strings.Builder
	switch mssg.Event {
	case "progress":
		err = app.RenderAssociated(&html, partials.ActivityProgress{Progress: mssg.Data})
	default:
		err = app.RenderAssociated(&html, partials.ActivityItem{Event: mssg.Event, Data: mssg.Data})
	}
	if err != nil {
		return ""
	}
	// trim multiple whitespaces from html
	return regexp.MustCompile(`\s+`).ReplaceAllString(html.String(), " ")
}

func (app *App) withError(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err == nil {
			return
		}
		log.Println("err:", err)
		message, desc, status := "Something went wrong!", "", http.StatusInternalServerError
		if cerr, ok := err.(ClientError); ok {
			message, desc, status = cerr.Message, cerr.Desc, cerr.Status
		}
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Add("HX-Retarget", "#notifications")
			w.Header().Add("HX-Reswap", "afterbegin")
			app.Render(w, partials.AlertError(message, desc))
		} else {
			w.WriteHeader(status)
			err := app.Render(w, pages.ErrorPage{Message: message, Desc: desc})
			if err != nil {
				log.Println(err)
				w.Write([]byte("Something went wrong!"))
			}
		}
	}
}

func detectContentType(file io.ReadSeeker) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return "", err
	}
	buf = buf[:n]
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	return http.DetectContentType(buf), nil
}
