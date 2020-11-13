/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

// this elements.go file provides the helper funtions to manipulate the xmpp elements as specified
// in the xep-0045 specification

import (
	"github.com/google/uuid"
	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func getRoomUpdatedElement(nonAnonymous, updatedAnonimity bool) *xmpp.Element {
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(newStatusElement("104"))
	if updatedAnonimity {
		if nonAnonymous {
			xEl.AppendElement(newStatusElement("172"))
		} else {
			xEl.AppendElement(newStatusElement("173"))
		}
	}
	msgEl := xmpp.NewElementName("message").SetID(uuid.New().String()).SetType("groupchat")
	return msgEl.AppendElement(xEl)
}

func getOccupantsInfoElement(occupants []*mucmodel.Occupant, id string,
	includeUserJID bool) *xmpp.Element {
	query := xmpp.NewElementNamespace("query", mucNamespaceAdmin)
	for _, o := range occupants {
		query.AppendElement(newOccupantItem(o, includeUserJID, true))
	}
	iq := xmpp.NewElementName("iq").AppendElement(query)
	iq.SetID("id").SetType("result")
	return iq
}

func getUserBannedElement(actor, reason string) *xmpp.Element {
	actorEl := xmpp.NewElementName("actor").SetAttribute("nick", actor)
	itemEl := xmpp.NewElementName("item").AppendElement(actorEl)
	itemEl.SetAttribute("affiliation", "outcast")
	itemEl.SetAttribute("role", "none")
	if reason != "" {
		reasonEl := xmpp.NewElementName("reason").SetText(reason)
		itemEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(itemEl)
	xEl.AppendElement(newStatusElement("301"))
	presence := xmpp.NewElementName("presence").SetType("unavailable").AppendElement(xEl)
	return presence
}

func getRoomMemberRemovedElement(actor, reason string) *xmpp.Element {
	actorEl := xmpp.NewElementName("actor").SetAttribute("nick", actor)
	itemEl := xmpp.NewElementName("item").AppendElement(actorEl)
	itemEl.SetAttribute("affiliation", "none")
	itemEl.SetAttribute("role", "none")
	if reason != "" {
		reasonEl := xmpp.NewElementName("reason").SetText(reason)
		itemEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(itemEl)
	xEl.AppendElement(newStatusElement("321"))
	presence := xmpp.NewElementName("presence").SetType("unavailable").AppendElement(xEl)
	return presence
}

// getReasonFromItem returns text from the reason element if specified, otherwise empty string
func getReasonFromItem(item xmpp.XElement) string {
	reasonEl := item.Elements().Child("reason")
	reason := ""
	if reasonEl != nil {
		reason = reasonEl.Text()
	}
	return reason
}

func getOccupantChangeElement(o *mucmodel.Occupant, reason string) *xmpp.Element {
	itemEl := xmpp.NewElementName("item")
	itemEl.SetAttribute("affiliation", o.GetAffiliation())
	itemEl.SetAttribute("role", o.GetRole())
	itemEl.SetAttribute("nick", o.OccupantJID.Resource())
	if reason != "" {
		reasonEl := xmpp.NewElementName("reason").SetText(reason)
		itemEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(itemEl)
	return xmpp.NewElementName("presence").AppendElement(xEl)
}

func getKickedOccupantElement(actor, reason string, selfNotifying bool) *xmpp.Element {
	itemEl := xmpp.NewElementName("item").SetAttribute("affiliation", "none")
	itemEl.SetAttribute("role", "none")
	actorEl := xmpp.NewElementName("actor").SetAttribute("nick", actor)
	itemEl.AppendElement(actorEl)
	if reason != "" {
		reasonEl := xmpp.NewElementName("reason").SetText(reason)
		itemEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(itemEl)
	xEl.AppendElement(newStatusElement("307"))
	if selfNotifying {
		xEl.AppendElement(newStatusElement("110"))
	}
	pEl := xmpp.NewElementName("presence").SetType("unavailable").AppendElement(xEl)
	return pEl
}

// getInvitedUserJID returns jid as specified in the "to" attribute of the invite element
func getInvitedUserJID(message *xmpp.Message) *jid.JID {
	invJIDStr := message.Elements().Child("x").Elements().Child("invite").Attributes().Get("to")
	invJID, _ := jid.NewWithString(invJIDStr, true)
	return invJID
}

func getMessageElement(body xmpp.XElement, id string, private bool) *xmpp.Element {
	msgEl := xmpp.NewElementName("message").AppendElement(body)

	if id != "" {
		msgEl.SetID(id)
	} else {
		msgEl.SetID(uuid.New().String())
	}

	if private {
		msgEl.SetType("chat")
		msgEl.AppendElement(xmpp.NewElementNamespace("x", mucNamespaceUser))
	} else {
		msgEl.SetType("groupchat")
	}

	return msgEl
}

func getDeclineStanza(room *mucmodel.Room, message *xmpp.Message) xmpp.Stanza {
	toStr := message.Elements().Child("x").Elements().Child("decline").Attributes().Get("to")
	to, _ := jid.NewWithString(toStr, true)

	declineEl := xmpp.NewElementName("decline").SetAttribute("from",
		message.FromJID().ToBareJID().String())
	reasonEl := message.Elements().Child("x").Elements().Child("decline").Elements().Child("reason")
	if reasonEl != nil {
		declineEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(declineEl)
	msgEl := xmpp.NewElementName("message").AppendElement(xEl).SetID(message.ID())
	msg, err := xmpp.NewMessageFromElement(msgEl, room.RoomJID, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	return msg
}

func getInvitationStanza(room *mucmodel.Room, inviteFrom, inviteTo *jid.JID, message *xmpp.Message) xmpp.Stanza {
	inviteEl := xmpp.NewElementName("invite").SetAttribute("from", inviteFrom.String())
	reasonEl := message.Elements().Child("x").Elements().Child("invite").Elements().Child("reason")
	if reasonEl != nil {
		inviteEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(inviteEl)
	if room.Config.PwdProtected {
		pwdEl := xmpp.NewElementName("password").SetText(room.Config.Password)
		xEl.AppendElement(pwdEl)
	}
	msgEl := xmpp.NewElementName("message").AppendElement(xEl).SetID(message.ID())
	msg, err := xmpp.NewMessageFromElement(msgEl, room.RoomJID, inviteTo)
	if err != nil {
		log.Error(err)
		return nil
	}
	return msg
}

func getOccupantUnavailableElement(o *mucmodel.Occupant, selfNotifying,
	includeUserJID bool) *xmpp.Element {
	// get the x element
	x := xmpp.NewElementNamespace("x", mucNamespaceUser)
	x.AppendElement(newOccupantItem(o, includeUserJID, true))
	x.AppendElement(newStatusElement("303"))
	if selfNotifying {
		x.AppendElement(newStatusElement("110"))
	}

	el := xmpp.NewElementName("presence").AppendElement(x).SetID(uuid.New().String())
	el.SetType("unavailable")
	return el
}

func getPasswordFromPresence(presence *xmpp.Presence) string {
	x := presence.Elements().ChildNamespace("x", mucNamespace)
	if x == nil {
		return ""
	}
	pwd := x.Elements().Child("password")
	if pwd == nil {
		return ""
	}
	return pwd.Text()
}

func getOccupantStatusElement(o *mucmodel.Occupant, selfNotifying,
	includeUserJID bool) *xmpp.Element {
	x := newOccupantAffiliationRoleElement(o, includeUserJID, false)
	if selfNotifying {
		x.AppendElement(newStatusElement("110"))
	}
	el := xmpp.NewElementName("presence").AppendElement(x).SetID(uuid.New().String())
	return el
}

func getOccupantSelfPresenceElement(o *mucmodel.Occupant, nonAnonymous bool,
	id string) *xmpp.Element {
	x := newOccupantAffiliationRoleElement(o, false, false)
	x.AppendElement(newStatusElement("110"))
	if nonAnonymous {
		x.AppendElement(newStatusElement("100"))
	}
	return xmpp.NewElementName("presence").AppendElement(x).SetID(id)
}

func getRoomSubjectElement(subject string) *xmpp.Element {
	s := xmpp.NewElementName("subject").SetText(subject)
	m := xmpp.NewElementName("message").SetType("groupchat").SetID(uuid.New().String())
	return m.AppendElement(s)
}

func getAckStanza(from, to *jid.JID) xmpp.Stanza {
	item := xmpp.NewElementName("item")
	item.SetAttribute("affiliation", "owner").SetAttribute("role", "moderator")
	e := xmpp.NewElementNamespace("x", mucNamespaceUser)
	e.AppendElement(item)
	e.AppendElement(newStatusElement("110"))
	e.AppendElement(newStatusElement("210"))

	presence := xmpp.NewElementName("presence").AppendElement(e)
	ack, err := xmpp.NewPresenceFromElement(presence, from, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	return ack
}

func getFormStanza(iq *xmpp.IQ, form *xep0004.DataForm) xmpp.Stanza {
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner)
	query.AppendElement(form.Element())

	e := xmpp.NewElementName("iq").SetID(iq.ID()).SetType("result").AppendElement(query)
	stanza, err := xmpp.NewIQFromElement(e, iq.ToJID(), iq.FromJID())
	if err != nil {
		log.Error(err)
		return nil
	}
	return stanza
}

func newStatusElement(code string) *xmpp.Element {
	s := xmpp.NewElementName("status")
	s.SetAttribute("code", code)
	return s
}

func newOccupantItem(o *mucmodel.Occupant, includeUserJID, includeNick bool) *xmpp.Element {
	i := xmpp.NewElementName("item")
	a := o.GetAffiliation()
	r := o.GetRole()
	if a == "" {
		a = "none"
	}
	if r == "" {
		r = "none"
	}
	i.SetAttribute("affiliation", a)
	i.SetAttribute("role", r)
	if includeUserJID {
		i.SetAttribute("jid", o.BareJID.String())
	}
	if includeNick {
		i.SetAttribute("nick", o.OccupantJID.Resource())
	}
	return i
}

func newOccupantAffiliationRoleElement(o *mucmodel.Occupant, includeUserJID,
	includeNick bool) *xmpp.Element {
	item := newOccupantItem(o, includeUserJID, includeNick)
	e := xmpp.NewElementNamespace("x", mucNamespaceUser)
	e.AppendElement(item)
	return e
}

// addResourceToBareJID joins bareJID and resource into a full jid
func addResourceToBareJID(bareJID *jid.JID, resource string) *jid.JID {
	res, _ := jid.NewWithString(bareJID.String()+"/"+resource, true)
	return res
}