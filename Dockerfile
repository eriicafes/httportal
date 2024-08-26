FROM golang:1.22.2 as build
WORKDIR /app
# install deps
COPY go.mod go.sum ./
RUN go mod download
# copy source files
COPY . .
# run build
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/app

FROM node:22-alpine as node-build
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable
WORKDIR /app
# install deps
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
# copy source files
COPY . .
# run build
RUN pnpm run build

FROM alpine:latest
WORKDIR /app
ENV NODE_ENV="production"
ENV PORT="8080"
COPY --from=build /app/bin/app .
COPY --from=build /app/views views
COPY --from=node-build /app/dist dist
EXPOSE 8080

# run
CMD ["./app"]
