package utils

import "errors"

var (
	ConnectionLossError = errors.New("connection loss")

	SendMessageFull = errors.New("send message full")

	JoinRoomTwice = errors.New("join room twice")

	NotInRoom = errors.New("not in room")

	RoomIdInvalid = errors.New("room id invalid")

	DisPatchChannelFull = errors.New("dispatch channel full")

	MergeChannelFull = errors.New("merge channel full")

	CertInvalid = errors.New("cert invalid")

	LogicDisPatchChannelFull = errors.New("logic dispatch channel full")
)
