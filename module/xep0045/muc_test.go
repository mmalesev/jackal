/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"crypto/tls"
	"testing"
	"context"

	c2srouter "github.com/ortuman/jackal/c2s/router"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/host"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
	"github.com/ortuman/jackal/stream"
	"github.com/pborman/uuid"
	"github.com/ortuman/jackal/module/xep0004"
)

func TestXEP0045_NewService(t *testing.T) {
	r, c := setupTest("jackal.im")

	failedMuc := New(&Config{MucHost: "jackal.im"}, nil, c, r)
	require.Nil(t, failedMuc)

	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	require.False(t, muc.router.Hosts().IsConferenceHost("jackal.im"))
	require.True(t, muc.router.Hosts().IsConferenceHost("conference.jackal.im"))

	require.Equal(t, muc.GetMucHostname(), "conference.jackal.im")
}

func TestXEP0045_NewRoomFromPresence(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	e := xmpp.NewElementNamespace("x", mucNamespace)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	muc.ProcessPresence(context.Background(), presence)

	// sender receives the appropriate response
	ack := stm.ReceiveElement()
	require.Equal(t, ack.String(), getAckStanza(to, from).String())

	// the room is created
	roomMem, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, roomMem)
	require.Equal(t, to.ToBareJID().String(), roomMem.RoomJID.String())
	require.Equal(t, muc.allRooms[0].RoomJID.String(), to.ToBareJID().String())
	oMem, err := c.Occupant().FetchOccupant(nil, to)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	require.Equal(t, from.String(), oMem.FullJID.String())
	//make sure the room is locked
	require.True(t, roomMem.Locked)
}

func TestXEP0045_NewInstantRoomFromIQ(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// creating a locked room
	err := muc.newRoom(context.Background(), from, to, "room", "nick", true)
	require.Nil(t, err)
	room, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.True(t, room.Locked)

	// instant room create iq
	x := xmpp.NewElementNamespace("x", dataNamespace).SetAttribute("type", "submit")
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner).AppendElement(x)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("set").AppendElement(query)
	request, err := xmpp.NewIQFromElement(iq, from, to)
	require.Nil(t, err)

	// sending an instant room request into the stream
	require.True(t, muc.MatchesIQ(request))
	muc.ProcessIQ(context.Background(), request)

	// receive the instant room creation confirmation
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)
	require.Equal(t, ack, request.ResultIQ())

	// the room should be unlocked now
	updatedRoom, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.False(t, updatedRoom.Locked)
}

func TestXEP0045_LegacyGroupchatRoomFromPresence(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// no <x> element, in order to support legacy groupchat 1.0
	p := xmpp.NewElementName("presence")
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	muc.ProcessPresence(context.Background(), presence)

	// sender receives the appropriate response
	ack := stm.ReceiveElement()
	require.Equal(t, ack.String(), getAckStanza(to, from).String())

	// the room is created
	roomMem, _ := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Equal(t, to.ToBareJID().String(), roomMem.RoomJID.String())
	//make sure the room is NOT locked (this is the only difference from MUC)
	require.False(t, roomMem.Locked)
}

func TestXEP0045_NewReservedRoomGetConfig(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// creating a locked room
	err := muc.newRoom(context.Background(), from, to, "room", "nick", true)
	require.Nil(t, err)
	room, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.True(t, room.Locked)

	// request configuration form
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("get").AppendElement(query)
	request, err := xmpp.NewIQFromElement(iq, from, to)
	require.Nil(t, err)

	// sending an instant room request into the stream
	require.True(t, muc.MatchesIQ(request))
	muc.ProcessIQ(context.Background(), request)

	// receive the room configuration form
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)
	require.Equal(t, ack.From(), to.String())
	require.Equal(t, ack.To(), from.String())
	require.Equal(t, ack.Name(), "iq")
	require.Equal(t, ack.Type(), "result")
	require.Equal(t, ack.ID(), "create1")

	queryResult := ack.Elements().Child("query")
	require.NotNil(t, queryResult)
	require.Equal(t, queryResult.Namespace(), mucNamespaceOwner)

	formElement := queryResult.Elements().Child("x")
	require.NotNil(t, formElement)
	form, err := xep0004.NewFormFromElement(formElement)
	require.Nil(t, err)
	require.Equal(t, form.Type, xep0004.Form)
	// the total number of fields should be 23
	require.Equal(t, len(form.Fields), 23)
}

func setupTest(domain string) (router.Router, repository.Container) {
	hosts, _ := host.New([]host.Config{{Name: domain, Certificate: tls.Certificate{}}})
	rep, _ := memorystorage.New()
	r, _ := router.New(
		hosts,
		c2srouter.New(rep.User(), memorystorage.NewBlockList()),
		nil,
	)
	return r, rep
}