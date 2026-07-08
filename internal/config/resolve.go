package config

//These values will be used when there is no CLI input or config file is empty.
//There is nothing written in the Assessment document about the default value ,
// the assessment document only mentions the order. I am taking assumption on these values
const (
	defaultToleranceCents            int64  = 0
	defaultMatchingWindowDays        int    = 0
	defaultOutputFormat              string = "jsonl"
	defaultPartitionBy               string = ""
	defaultIncludePendingSettlements bool   = false
	defaultQueryLimit                int    = 100
)

//I have defined the type of the fields as pointers
//so that in case CLI does not pass any value for any of the fields - a nil pointer will be assigned.
//

type CLIOverrides struct {
	ToleranceCents            *int64
	MatchingWindowDays        *int
	OutputFormat              *string
	PartitionBy               *string
	IncludePendingSettlements *bool
	QueryLimit                *int
	Filters                   map[string]string
}

type Resolver struct {
	cli CLIOverrides
	cfg rawConfig
}

func NewResolver(cli CLIOverrides, cfg rawConfig) Resolver {
	return Resolver{cli: cli, cfg: cfg}
}

// Here I am  checking which value to pass as configuration - the CLI values or the Config.json values or default values
func (r Resolver) ToleranceCents() int64 {
	if r.cli.ToleranceCents != nil {
		return *r.cli.ToleranceCents
	}
	if r.cfg.ToleranceCents != nil {
		return *r.cfg.ToleranceCents
	}
	return defaultToleranceCents
}

func (r Resolver) MatchingWindowDays() int {
	if r.cli.MatchingWindowDays != nil {
		return *r.cli.MatchingWindowDays
	}
	if r.cfg.MatchingWindowDays != nil {
		return *r.cfg.MatchingWindowDays
	}
	return defaultMatchingWindowDays
}

func (r Resolver) OutputFormat() string {
	if r.cli.OutputFormat != nil {
		return *r.cli.OutputFormat
	}
	if r.cfg.OutputFormat != nil {
		return *r.cfg.OutputFormat
	}
	return defaultOutputFormat
}

func (r Resolver) PartitionBy() string {
	if r.cli.PartitionBy != nil {
		return *r.cli.PartitionBy
	}
	if r.cfg.PartitionBy != nil {
		return *r.cfg.PartitionBy
	}
	return defaultPartitionBy
}

func (r Resolver) IncludePendingSettlements() bool {
	if r.cli.IncludePendingSettlements != nil {
		return *r.cli.IncludePendingSettlements
	}
	if r.cfg.IncludePendingSettlements != nil {
		return *r.cfg.IncludePendingSettlements
	}
	return defaultIncludePendingSettlements
}

func (r Resolver) QueryLimit() int {
	if r.cli.QueryLimit != nil {
		return *r.cli.QueryLimit
	}
	if r.cfg.QueryLimit != nil {
		return *r.cfg.QueryLimit
	}
	return defaultQueryLimit
}

//I've created the combined filter for config+cmd here. If the config and cmd are updating the same filter , cmd will take precedence
//in the second for loop I've updated the filters value from cli.Filters
func (r Resolver) Filters() map[string]any {
	merged := make(map[string]any, len(r.cfg.Filters)+len(r.cli.Filters))
	for k, v := range r.cfg.Filters {
		merged[k] = v
	}
	for k, v := range r.cli.Filters {
		merged[k] = v
	}

	return merged
}
