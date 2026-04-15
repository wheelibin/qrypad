package db

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	DataFetchedMsg           QueryResultMsg
	TableInfoDataFetchedMsg  QueryResultMsg
	SchemaEntitiesFetchedMsg QueryResultMsg
	DatabaseListFetchedMsg   QueryResultMsg
)
