package model

import "encoding/json"

type Request struct {
	Type            string
	RequireResponse bool
}

var (
	R_NAME = Request{
		Type:            "NAME",
		RequireResponse: true,
	}
	R_TALK = Request{
		Type:            "TALK",
		RequireResponse: true}
	R_WHISPER = Request{
		Type:            "WHISPER",
		RequireResponse: true}
	R_VOTE = Request{
		Type:            "VOTE",
		RequireResponse: true}
	R_DIVINE = Request{
		Type:            "DIVINE",
		RequireResponse: true}
	R_GUARD = Request{
		Type:            "GUARD",
		RequireResponse: true}
	R_ATTACK = Request{
		Type:            "ATTACK",
		RequireResponse: true}
	R_INITIALIZE = Request{
		Type:            "INITIALIZE",
		RequireResponse: false}
	R_DAILY_INITIALIZE = Request{
		Type:            "DAILY_INITIALIZE",
		RequireResponse: false}
	R_DAILY_FINISH = Request{
		Type:            "DAILY_FINISH",
		RequireResponse: false}
	R_FINISH = Request{
		Type:            "FINISH",
		RequireResponse: false}
	R_TALK_PHASE_START = Request{
		Type:            "TALK_PHASE_START",
		RequireResponse: false}
	R_TALK_PHASE_END = Request{
		Type:            "TALK_PHASE_END",
		RequireResponse: false}
	R_TALK_BROADCAST = Request{
		Type:            "TALK_BROADCAST",
		RequireResponse: false}
	R_WHISPER_PHASE_START = Request{
		Type:            "WHISPER_PHASE_START",
		RequireResponse: false}
	R_WHISPER_PHASE_END = Request{
		Type:            "WHISPER_PHASE_END",
		RequireResponse: false}
	R_WHISPER_BROADCAST = Request{
		Type:            "WHISPER_BROADCAST",
		RequireResponse: false}
)

func (r Request) String() string {
	return r.Type
}

func (r Request) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func RequestFromString(s string) Request {
	switch s {
	case "NAME":
		return R_NAME
	case "TALK":
		return R_TALK
	case "WHISPER":
		return R_WHISPER
	case "VOTE":
		return R_VOTE
	case "DIVINE":
		return R_DIVINE
	case "GUARD":
		return R_GUARD
	case "ATTACK":
		return R_ATTACK
	case "INITIALIZE":
		return R_INITIALIZE
	case "DAILY_INITIALIZE":
		return R_DAILY_INITIALIZE
	case "DAILY_FINISH":
		return R_DAILY_FINISH
	case "FINISH":
		return R_FINISH
	case "TALK_PHASE_START":
		return R_TALK_PHASE_START
	case "TALK_PHASE_END":
		return R_TALK_PHASE_END
	case "TALK_BROADCAST":
		return R_TALK_BROADCAST
	case "WHISPER_PHASE_START":
		return R_WHISPER_PHASE_START
	case "WHISPER_PHASE_END":
		return R_WHISPER_PHASE_END
	case "WHISPER_BROADCAST":
		return R_WHISPER_BROADCAST
	}
	return Request{}
}
