package model

//This defines an entry that will be written in exceptions file.

type Exceptions struct {

	//These two fields are mentioned in the pdf as query parameters for exception table.
	ExceptionType string
	Severity      string

	//These are the assumptions I am making based on the requirement that I have to validate
	//required colums in every file and reject malformed rows with source file, line number
	//and reason.

	SourceFile string
	Reason     string
	LineNumber int

	// adding additional ID , this will reference to ID of the record from respective table.
	//For eg. etry_id if exception is from ledger_entries.csv , statement_id if exception is from
	// bank_settlements.csv etc

	RecordID string
}
