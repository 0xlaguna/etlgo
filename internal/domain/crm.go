package domain

import (
	"time"
)

type OpportunityStage string

const (
	StageLead        OpportunityStage = "lead"
	StageOpportunity OpportunityStage = "opportunity"
	StageClosedWon   OpportunityStage = "closed_won"
	StageClosedLost  OpportunityStage = "closed_lost"
)

type Opportunity struct {
	OpportunityID string           `json:"opportunity_id"`
	ContactEmail  string           `json:"contact_email"`
	Stage         OpportunityStage `json:"stage"`
	Amount        float64          `json:"amount"`
	CreatedAt     string           `json:"created_at"`
	UTMCampaign   string           `json:"utm_campaign"`
	UTMSource     string           `json:"utm_source"`
	UTMMedium     string           `json:"utm_medium"`
}

type CRMData struct {
	External struct {
		CRM struct {
			Opportunities []Opportunity `json:"opportunities"`
		} `json:"crm"`
	} `json:"external"`
}

type ProcessedOpportunity struct {
	OpportunityID string           `json:"opportunity_id"`
	ContactEmail  string           `json:"contact_email"`
	Stage         OpportunityStage `json:"stage"`
	Amount        float64          `json:"amount"`
	CreatedAt     time.Time        `json:"created_at"`
	UTMCampaign   string           `json:"utm_campaign"`
	UTMSource     string           `json:"utm_source"`
	UTMMedium     string           `json:"utm_medium"`
	ProcessedAt   time.Time        `json:"processed_at"`
}

func (o ProcessedOpportunity) IsLead() bool {
	return o.Stage == StageLead
}

// true if the opportunity is in opportunity stage
func (o ProcessedOpportunity) IsOpportunity() bool {
	return o.Stage == StageOpportunity
}

// returns true if the opportunity is closed won
func (o ProcessedOpportunity) IsClosedWon() bool {
	return o.Stage == StageClosedWon
}

// returns true if the opportunity is closed lost
func (o ProcessedOpportunity) IsClosedLost() bool {
	return o.Stage == StageClosedLost
}
