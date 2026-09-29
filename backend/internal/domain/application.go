package domain

import (
	"strings"
	"time"
)

type ApplicationStatus string

const (
	ApplicationStatusPending   ApplicationStatus = "pending"
	ApplicationStatusConfirmed ApplicationStatus = "confirmed"
	ApplicationStatusRejected  ApplicationStatus = "rejected"
	ApplicationStatusCancelled ApplicationStatus = "cancelled"
)

type Application struct {
	ID          int               `json:"id"`
	SectionID   int               `json:"sectionId"`
	UserID      int64             `json:"userId"`
	ParentName  string            `json:"parentName"`
	ParentPhone string            `json:"parentPhone"`
	ChildName   string            `json:"childName"`
	ChildAge    int               `json:"childAge"`
	Status      ApplicationStatus `json:"status"`
	CreatedAt   time.Time         `json:"createdAt"`
}

func (application Application) IsPending() bool {
	return application.Status == ApplicationStatusPending
}

func (application Application) IsCancelled() bool {
	return application.Status == ApplicationStatusCancelled
}

func (application Application) HasParentPhone() bool {
	return strings.TrimSpace(application.ParentPhone) != ""
}

func (application Application) MaskedParentPhone() string {
	digits := digitsOnly(application.ParentPhone)
	if len(digits) < 2 {
		return "***-**-**"
	}
	return "***-**-" + digits[len(digits)-2:]
}

func digitsOnly(source string) string {
	var builder strings.Builder
	for _, symbol := range source {
		if symbol >= '0' && symbol <= '9' {
			builder.WriteRune(symbol)
		}
	}
	return builder.String()
}
