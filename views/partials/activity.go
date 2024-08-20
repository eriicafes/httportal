package partials

type ActivityConnector struct {
	ID string
}

func (t ActivityConnector) AssociatedTemplate() (string, string, any) {
	return "partials/activity", "activity-connector", t
}

type ActivityProgress struct {
	Progress string
}

func (t ActivityProgress) AssociatedTemplate() (string, string, any) {
	return "partials/activity", "activity-progress", t
}

type ActivityItem struct {
	Event string
	Data  string
}

func (t ActivityItem) AssociatedTemplate() (string, string, any) {
	return "partials/activity", "activity-item", t
}

type Activity struct {
	ID string
}

func (t Activity) Template() (string, any) {
	return "partials/activity", t
}
