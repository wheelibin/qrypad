package db

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

type Data struct {
	Columns   []string
	Rows      []map[string]any
	QueryTime time.Duration
}

type (
	DatabaseConnectedMsg DBConn
	QueryResultMsg       struct {
		Data *Data
		Err  error
	}
	QueryControlMsg struct {
		Cancel     context.CancelFunc
		ResultChan <-chan tea.Msg
	}
	DataFetchedMsg             QueryResultMsg
	TableInfoDataFetchedMsg    QueryResultMsg
	SchemaEntitiesFetchedMsg   QueryResultMsg
	DatabaseListFetchedMsg     QueryResultMsg
	ConnectionListFetchedMsg   QueryResultMsg
	AutoCompleteDataFetchedMsg []string
)
