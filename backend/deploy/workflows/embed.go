package workflows

import _ "embed"

// TicketProcess is the default BPMN model for the ticket workflow.
//
//go:embed ticket-process.bpmn
var TicketProcess []byte
