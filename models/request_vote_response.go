package models

type RequestVoteResponse struct {
	Term        int
	VoteGranted bool
}