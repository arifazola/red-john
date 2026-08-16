package models

type RequestVoteRequest struct {
	Term        int
	CandidateID string
}