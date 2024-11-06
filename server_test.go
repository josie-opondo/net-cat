package main

import (
	"net"
	"testing"
)

func TestServer_Logo(t *testing.T) {
    type fields struct {
        listenAddr string
        ln         net.Listener
        msgChan    chan Message
        clients    map[net.Conn]string
        sem        chan struct{}
        msgStore   []Message
        shutdown   chan struct{}
    }
    tests := []struct {
        name    string
        fields  fields
        want    string
        wantErr bool
    }{
        {
            name: "Valid Logo",
            fields: fields{
                listenAddr: ":8080",
                ln:         nil,
                msgChan:    make(chan Message),
                clients:    make(map[net.Conn]string),
                sem:        make(chan struct{}, 10),
                msgStore:   []Message{},
                shutdown:   make(chan struct{}),
            },
            want: "\033[34m" + // Start blue background
			"          _nnnn_\n" +
			"         \033[32mdGGGGMMb\033[34m\n" + // Green
			"        \033[32m@p~qp~~qMb\033[34m\n" + // Green
			"        \033[32mM|\033[33m@\033[32m||\033[33m@) M|\033[34m\n" + // Green with yellow for '@'
			"        \033[32m@,----.JM|\033[34m\n" + // Green
			"       \033[32mJS^\\__/  qKL\033[34m\n" + // Green
			"      \033[32mdZP        qKRb\033[34m\n" + // Green
			"     \033[32mdZP          qKKb\033[34m\n" + // Green
			"    \033[32mfZP            SMMb\033[34m\n" + // Green
			"    \033[32mHZM            MMMM\033[34m\n" + // Green
			"    \033[32mFqM            MMMM\033[34m\n" + // Green
			" \033[34m__\033[32m | \".        |\\dS\"qML\033[34m\n" + // Green with blue
			" \033[34m|    `.        | `' \\Zq\033[34m\n" +
			" \033[34m_)      \\.___.,|     .'\033[34m\n" +
			" \033[34m\\____   )MMMMMP|   .'\033[34m\n" +
			"      `-'       `--'\033[0m",
            wantErr: false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := &Server{
                listenAddr: tt.fields.listenAddr,
                ln:         tt.fields.ln,
                msgChan:    tt.fields.msgChan,
                clients:    tt.fields.clients,
                sem:        tt.fields.sem,
                msgStore:   tt.fields.msgStore,
                shutdown:   tt.fields.shutdown,
            }
            got, err := s.Logo()
            if (err != nil) != tt.wantErr {
                t.Errorf("Server.Logo() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Server.Logo() = %v, want %v", got, tt.want)
            }
        })
    }
}
