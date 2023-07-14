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

type Listener struct {
	indexer *rpc.Indexer
	sck     *websocket.Conn
	done    chan struct{}
}

func NewListener(host string, indexer *rpc.Indexer) *Listener {
	ws := &Listener{
		done:    make(chan struct{}),
		indexer: indexer,
	}

	c, _, err := websocket.DefaultDialer.Dial(host, nil)
	if err != nil {
		slog.Error("error while dialing websocket", "error", err)
	}
	slog.Info("connected", "host", host)

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

			go ws.indexer.ProcessBlock(bHash)
			slog.Info("recv", "num", num, "hash", bHash)
		}
	}()

	ws.sck = c

	return ws
}

func (ws *Listener) Subscribe() error {
	slog.Info("subscribing to newHeads")
	if err := ws.sck.WriteMessage(websocket.TextMessage, []byte(`{"id":1,"jsonrpc":"2.0","method":"eth_subscribe","params":["newHeads"]}`)); err != nil {
		return errors.Wrap(err, "failed to subscribe to newHeads")
	}
	return nil
}

func (ws *Listener) Done() <-chan struct{} {
	return ws.done
}

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
