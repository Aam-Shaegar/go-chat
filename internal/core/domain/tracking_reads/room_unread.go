package domain_trackingreads

type RoomUnread struct {
	RoomID string `db:"room_id" json:"room_id"`
	Unread int    `db:"unread" json:"unread"`
}

func NewRoomUnread(roomID string, unread int) *RoomUnread {
	return &RoomUnread{
		RoomID: roomID,
		Unread: unread,
	}
}

type DMUnread struct {
	FromUserID string `db:"from_user_id" json:"from_user_id"`
	Unread     int    `db:"unread" json:"unread"`
}

func NewDMUnread(fromUserID string, unread int) *DMUnread {
	return &DMUnread{
		FromUserID: fromUserID,
		Unread:     unread,
	}
}
