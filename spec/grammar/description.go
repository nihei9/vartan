package grammar

type Terminal struct {
	Number        int    `json:"number"`
	Name          string `json:"name"`
	Anonymous     bool   `json:"anonymous"`
	Pattern       string `json:"pattern"`
	Precedence    int    `json:"prec"`
	Associativity string `json:"assoc"`
}

type NonTerminal struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

type Production struct {
	Number        int    `json:"number"`
	LHS           int    `json:"lhs"`
	RHS           []int  `json:"rhs"`
	Precedence    int    `json:"prec"`
	Associativity string `json:"assoc"`
}

type Item struct {
	Production int `json:"production"`
	Dot        int `json:"dot"`
}

type Transition struct {
	Symbol int `json:"symbol"`
	State  int `json:"state"`
}

type Reduce struct {
	LookAhead  []int `json:"look_ahead"`
	Production int   `json:"production"`
}

type SRConflict struct {
	Symbol            int  `json:"symbol"`
	State             int  `json:"state"`
	Production        int  `json:"production"`
	AdoptedState      *int `json:"adopted_state"`
	AdoptedProduction *int `json:"adopted_production"`
	ResolvedBy        int  `json:"resolved_by"`
}

type RRConflict struct {
	Symbol            int `json:"symbol"`
	Production1       int `json:"production_1"`
	Production2       int `json:"production_2"`
	AdoptedProduction int `json:"adopted_production"`
	ResolvedBy        int `json:"resolved_by"`
}

type State struct {
	Number     int           `json:"number"`
	Kernel     []*Item       `json:"kernel"`
	Shift      []*Transition `json:"shift"`
	Reduce     []*Reduce     `json:"reduce"`
	GoTo       []*Transition `json:"goto"`
	SRConflict []*SRConflict `json:"sr_conflict"`
	RRConflict []*RRConflict `json:"rr_conflict"`
}

type Report struct {
	Terminals    []*Terminal    `json:"terminals"`
	NonTerminals []*NonTerminal `json:"non_terminals"`
	Productions  []*Production  `json:"productions"`
	States       []*State       `json:"states"`
}
