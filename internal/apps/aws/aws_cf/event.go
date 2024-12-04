package aws_cf

type StackStatusChangeEvent struct {
	StackID            string                               `json:"stack-id"`
	ClientRequestToken string                               `json:"client-request-token"`
	StatusDetails      StackStatusChangeEvent_StatusDetails `json:"status-details"`
}

type StackStatusChangeEvent_StatusDetails struct {
	Status         string `json:"status"`
	DetailedStatus string `json:"detailed-status"`
}
