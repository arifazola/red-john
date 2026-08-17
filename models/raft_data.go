package models

type RaftData struct {
	ElectionInterval int
	Role             string
	Term             int
	VotedFor         string
	TotalVote        int
}