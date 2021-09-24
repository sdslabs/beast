package api

import (
	"time"
)

type HTTPPlainResp struct {
	Message string `json:"message" example:"Messsage in response to your request"`
}

type HTTPErrorResp struct {
	Error string `json:"error" example:"Error occured while veifying the challenge."`
}

type HTTPAuthorizeResp struct {
	Token   string `json:"token" example:"YOUR_AUTHENTICATION_TOKEN"`
	Role    string `json:"role" example:"author"`
	Message string `json:"message" example:"Response message"`
}

type AvailableImagesResp struct {
	Message string   `json:"message" example:"Available Base images."`
	Images  []string `json:"images" example:"['ubuntu16.04', 'ubuntu18.04']"`
}

type PortsInUseResp struct {
	MinPortValue uint32   `json:"port_min_value" example:"10000"`
	MaxPortValue uint32   `json:"port_max_value" example:"20000"`
	PortsInUse   []uint32 `json:"ports_in_use" example:"[100001, 100003, 10010]"`
}

type ChallengeStatusResp struct {
	Name      string    `json:"name" example:"Web Challenge"`
	Status    string    `json:"status" example:"deployed"`
	UpdatedAt time.Time `json:"updated_at" example:"2018-12-31T22:20:08.948096189+05:30"`
}

type ChallengesResp struct {
	Message    string
	Challenges []string
}

type LogsInfoResp struct {
	Stdout string `json:"stdout" example:"[INFO] Challenge is starting to deploy"`
	Stderr string `json:"stderr" example:"[ERROR] Challenge deployment failed."`
}

type ChallengeDescriptionResp struct {
	Name   string `json:"name" example:"Web Challenge"`
	Author string `json:"author" example:"Fristonio"`
	Desc   string `json:"desc" example:"Challenge Description"`
}

type NotificationResp struct {
	ID        uint      `json:"id" example:"3"`
	Title     string    `json:"title" example:"CTF is live now!"`
	Desc      string    `json:"desc" example:"Notification Description"`
	UpdatedAt time.Time `json:"updated_at" example:"2018-12-31T22:20:08.948096189+05:30"`
}

type UserResp struct {
	Id         uint                 `json:"id" example:"5"`
	Username   string               `json:"username" example:"CTF is live now!"`
	Role       string               `json:"role" example:"author"`
	Status     uint                 `json:"status" example:"0"`
	Score      uint                 `json:"score" example:"750"`
	Rank       int64                `json:"rank" example:"15"`
	Email      string               `json:"email" example:"fristonio@gmail.com"`
	Challenges []ChallengeSolveResp `json:"challenges"`
}

type UsersResp struct {
	Id       uint   `json:"id" example:"5"`
	Username string `json:"username" example:"CTF is live now!"`
	Role     string `json:"role" example:"author"`
	Status   uint   `json:"status" example:"0"`
	Score    uint   `json:"score" example:"750"`
	Email    string `json:"email" example:"fristonio@gmail.com"`
}

type ChallengeSolveResp struct {
	Id       uint      `json:"id" example:"4"`
	Name     string    `json:"name" example:"Web Challenge"`
	Category string    `json:"category" example:"bare"`
	Tags     []string  `json:"tags" example:"['pwn','misc']"`
	SolvedAt time.Time `json:"solvedAt"`
	Points   uint      `json:"points" example:"50"`
}

type UserSolveResp struct {
	UserID   uint      `json:"id" example:"5"`
	Username string    `json:"username" example:"fristonio"`
	SolvedAt time.Time `json:"solvedAt"`
}

type ChallengeInfoResp struct {
	Name         string          `json:"name" example:"Web Challenge"`
	ChallId      uint            `json:"id" example:"0"`
	Category     string          `json:"category" example:"bare"`
	CreatedAt    time.Time       `json:"createdAt"`
	Status       string          `json:"status" example:"deployed"`
	Ports        []uint32        `json:"ports" example:"[3001, 3002]"`
	Hints        string          `json:"hints" example:"Try robots"`
	Desc         string          `json:"description" example:"A simple web challenge"`
	Points       uint            `json:"points" example:"50"`
	SolvesNumber int             `json:"solvesNumber" example:"100"`
	Solves       []UserSolveResp `json:"solves"`
}

type ChallengePreviewResp struct {
	Name     string   `json:"name" example:"Web Challenge"`
	Category string   `json:"category" example:"web"`
	Ports    []uint32 `json:"ports" example:"[3001, 3002]"`
	Hints    []string `json:"hints" example:"Try robots"`
	Desc     string   `json:"description" example:"A simple web challenge"`
	Points   uint     `json:"points" example:"50"`
}

type SubmissionResp struct {
	UserId    uint      `json:"user_id" example:"3"`
	Username  string    `json:"username" example:"fristonio"`
	ChallId   uint      `json:"chall_id" example:"3"`
	ChallName string    `json:"name" example:"Web Challenge"`
	Category  string    `json:"category" example:"web"`
	Points    uint      `json:"points" example:"50"`
	SolvedAt  time.Time `json:"solvedAt"`
}

type FlagSubmitResp struct {
	Message string `json:"message" example:"Your answer is correct"`
	Success bool   `json:"success" example:"true"`
}

type UsersStatisticsResp struct {
	TotalRegisteredUsers uint `json:"total_registered_users" example:"120"`
	BannedUsers          uint `json:"banned_users" example:"60"`
	UnbannedUsers        uint `json:"unbanned_users" example:"60"`
}

type CompetitionInfoResp struct {
	Name         string `json:"name" example:"fristonio"`
	About        string `json:"about" example:"This is a CTF competition"`
	Prizes       string `json:"prizes" example:"1st and 2nd place winners will get $10K"`
	StartingTime string `json:"starting_time"`
	EndingTime   string `json:"ending_time"`
	TimeZone     string `json:"timezone" example:"Asia/Calcutta: UTC +05:30"`
	LogoURL      string `json:"logo_url"`
}
