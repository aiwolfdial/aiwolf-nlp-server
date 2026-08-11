package model

import "github.com/aiwolfdial/aiwolf-nlp-server/model/wire"

type Vote struct {
	Day    int
	Agent  Agent
	Target Agent
}

func (v Vote) wire() wire.Vote {
	return wire.Vote{
		Day:    v.Day,
		Agent:  v.Agent.String(),
		Target: v.Target.String(),
	}
}

func wireVotes(votes []Vote) []wire.Vote {
	if len(votes) == 0 {
		return nil
	}
	out := make([]wire.Vote, 0, len(votes))
	for _, v := range votes {
		out = append(out, v.wire())
	}
	return out
}
