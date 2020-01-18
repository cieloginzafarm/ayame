package main

var (
	// register/unregister は待たせる
	registerChannel   = make(chan *register)
	unregisterChannel = make(chan *unregister)
	// ブロックされたくないので 100 に設定
	forwardChannel = make(chan forward, 100)
)

// roomId がキーになる
type room struct {
	clients map[string]*client
}

func server() {
	// room を管理するマップはここに用意する
	var m = make(map[string]room)
	// ここはシングルなのでロックは不要、多分
	for {
		select {
		case register := <-registerChannel:
			c := register.client
			rch := register.resultChannel
			r, ok := m[c.roomID]
			if ok {
				// room があった
				if len(r.clients) == 1 {
					// room に 自分を追加する
					// 登録しているのが同じ ID だった場合はエラーにする
					_, ok := r.clients[c.ID]
					if ok {
						// 重複エラー
						rch <- dup
					} else {
						r.clients[c.ID] = c
						m[c.roomID] = r
						rch <- two
					}
				} else {
					// room あったけど満杯
					rch <- full
				}
			} else {
				// room がなかった
				var clients = make(map[string]*client)
				clients[c.ID] = c
				// room を追加
				m[c.roomID] = room{
					clients: clients,
				}
				c.debugLog().Msg("CREATED-ROOM")
				rch <- one
			}
		case unregister := <-unregisterChannel:
			c := unregister.client
			// room を探す
			r, ok := m[c.roomID]
			// room がない場合は何もしない
			if ok {
				_, ok := r.clients[c.ID]
				if ok {
					for _, client := range r.clients {
						// 両方の forwardChannel を閉じる
						close(client.forwardChannel)
						client.debugLog().Msg("CLOSED-FORWARD-CHANNEL")
						client.debugLog().Msg("REMOVED-CLIENT")
					}
					// room を削除
					delete(m, c.roomID)
					c.debugLog().Msg("DELETED-ROOM")
				}
			}
		case forward := <-forwardChannel:
			r, ok := m[forward.client.roomID]
			// room がない場合は何もしない
			if ok {
				// room があった
				for clientID, client := range r.clients {
					// 自分ではない方に投げつける
					if clientID != forward.client.ID {
						client.forwardChannel <- forward
					}
				}
			}
		}
	}
}