package fetch

// ResultClass summarises how a scrape ended.
type ResultClass int

const (
	// ResultSuccess means the body arrived without any error.
	ResultSuccess ResultClass = iota
	// ResultLateSuccess means the body arrived together with a timeout error.
	ResultLateSuccess
	// ResultTimedOut means only an error came back.
	ResultTimedOut
	// ResultFailure means the transport failed for another reason.
	ResultFailure
)

// Classify turns a transport outcome into a stable result class.
func Classify(outcome Outcome) ResultClass {
	switch {
	case outcome.Response != nil && outcome.Err == nil:
		return ResultSuccess
	case outcome.Response != nil && outcome.Err != nil:
		return ResultLateSuccess
	case outcome.Err != nil:
		return ResultTimedOut
	default:
		return ResultFailure
	}
}
