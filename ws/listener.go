package ws

import (
	"avax-indexer/model"
	"avax-indexer/rpc"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"math/big"
	"time"
)

// Listener is a service that listens to newHeads events
type Listener struct {
	indexer *rpc.Indexer
	sck     *websocket.Conn
	done    chan struct{}
}

// NewListener initializes a new Listener service
// It dials the host and sets up the receiver goroutine
// For each received message it tries to unmarshal it into a newHead
// If successful, it starts a new goroutine to process the block
func NewListener(host string, indexer *rpc.Indexer) *Listener {
	// Create service
	ws := &Listener{
		done:    make(chan struct{}),
		indexer: indexer,
	}

	// Dial target ETH ws host
	c, _, err := websocket.DefaultDialer.Dial(host, nil)
	if err != nil {
		slog.Error("error while dialing websocket", "error", err)
	}
	slog.Info("connected to avax websocket feed", "host", host)

	// Set connection
	ws.sck = c

	// Start listener goroutine
	go func() {
		defer close(ws.done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if e := new(websocket.CloseError); errors.As(err, &e) {
					if e.Code == websocket.CloseNormalClosure {
						slog.Info("closed ws connection normally")
						return
					}
				}
				slog.Error("recv err", "error", err)
				return
			}

			var data model.WsResponse[model.NewHead]
			if err := json.Unmarshal(message, &data); err != nil {
				slog.Error("failed to unmarshal ws message", "error", err)
				return
			}

			if data.Method != "eth_subscription" {
				continue
			}

			bHash := data.Params.Result.Hash
			num := new(big.Int)
			fmt.Sscanf(data.Params.Result.Number, "0x%x", num)

			// Start processing goroutine
			go ws.indexer.ProcessBlock(bHash)
			slog.Info("recv", "num", num, "hash", bHash)
		}
	}()

	return ws
}

// Subscribe sends a subscription request for newHeads to the websocket
func (ws *Listener) Subscribe() error {
	slog.Info("subscribing to newHeads")
	if err := ws.sck.WriteMessage(websocket.TextMessage, []byte(`{"id":1,"jsonrpc":"2.0","method":"eth_subscribe","params":["newHeads"]}`)); err != nil {
		return errors.Wrap(err, "failed to subscribe to newHeads")
	}
	return nil
}

// Done returns a channel that is closed when the websocket connection is closed
func (ws *Listener) Done() <-chan struct{} {
	return ws.done
}

// GraceClose gracefully closes the websocket connection
func (ws *Listener) GraceClose() error {
	slog.Info("gracefully closing ws connection")

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := ws.sck.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return errors.Wrap(err, "failed to write close message")
	}

	select {
	case <-ws.done:
	case <-time.After(time.Second):
	}

	return nil
}
