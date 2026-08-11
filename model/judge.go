package model

import "github.com/aiwolfdial/aiwolf-nlp-server/model/wire"

type Judge struct {
	Day    int
	Agent  Agent
	Target Agent
	Result Species
}

func (j Judge) wire() wire.Judge {
	return wire.Judge{
		Day:    j.Day,
		Agent:  j.Agent.String(),
		Target: j.Target.String(),
		Result: wire.Species(j.Result),
	}
}

func wireJudge(judge *Judge) *wire.Judge {
	if judge == nil {
		return nil
	}
	j := judge.wire()
	return &j
}
