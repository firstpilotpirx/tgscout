package yaml

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml/ast"

	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

// Built-in catch-up depth when no level sets one.
const (
	DefaultMessages = 500
	DefaultDays     = 3
)

// depthPart is a catch-up depth as one level writes it: a field it leaves
// out is nil and comes from the level above.
type depthPart struct{ messages, days *int }

var builtInDepth = domain.CatchUp{Messages: DefaultMessages, Days: DefaultDays}

// over resolves this level's fields over the depth of the level above.
func (p depthPart) over(above domain.CatchUp) domain.CatchUp {
	if p.messages != nil {
		above.Messages = *p.messages
	}
	if p.days != nil {
		above.Days = *p.days
	}
	return above
}

func (w *walker) depth(node ast.Node, path string) depthPart {
	found, ok := w.fields(node, path, "messages", "days")
	if !ok {
		return depthPart{}
	}
	var part depthPart
	for key, target := range map[string]**int{"messages": &part.messages, "days": &part.days} {
		if entry := found[key]; entry != nil {
			if value, valid := w.number(entry.Value, join(path, key), 1); valid {
				*target = &value
			}
		}
	}
	return part
}

func (w *walker) requests(entry *ast.MappingValueNode, parentLine int, rootDepth domain.CatchUp) []domain.Request {
	if entry == nil || isNull(entry.Value) {
		w.fail(parentLine, "requests", "at least one request is needed")
		return nil
	}
	list, ok := entry.Value.(*ast.SequenceNode)
	if !ok {
		w.fail(lineOf(entry.Value), "requests", "must be a list")
		return nil
	}
	if len(list.Values) == 0 {
		w.fail(lineOf(entry.Key), "requests", "at least one request is needed")
		return nil
	}
	firstLine := map[string]int{}
	requests := make([]domain.Request, 0, len(list.Values))
	for i, item := range list.Values {
		request := w.request(item, fmt.Sprintf("requests[%d]", i), rootDepth)
		if line, seen := firstLine[request.ID]; seen && request.ID != "" {
			w.fail(lineOf(item), fmt.Sprintf("requests[%d].id", i),
				fmt.Sprintf("%q is used twice (first at line %d)", request.ID, line))
		} else if request.ID != "" {
			firstLine[request.ID] = lineOf(item)
		}
		requests = append(requests, request)
	}
	return requests
}

func (w *walker) request(node ast.Node, path string, rootDepth domain.CatchUp) domain.Request {
	found, ok := w.fields(node, path, "id", "find", "chats", "catch_up")
	if !ok {
		return domain.Request{}
	}
	request := domain.Request{
		ID:   w.requiredText(found["id"], join(path, "id"), lineOf(node)),
		Find: w.requiredText(found["find"], join(path, "find"), lineOf(node)),
	}
	requestDepth := rootDepth
	if catchUp := found["catch_up"]; catchUp != nil {
		requestDepth = w.depth(catchUp.Value, join(path, "catch_up")).over(rootDepth)
	}
	request.Chats = w.chats(found["chats"], join(path, "chats"), lineOf(node), requestDepth)
	return request
}

func (w *walker) requiredText(entry *ast.MappingValueNode, path string, parentLine int) string {
	if entry == nil || isNull(entry.Value) {
		w.fail(parentLine, path, "is required")
		return ""
	}
	text, ok := w.text(entry.Value, path)
	if ok && strings.TrimSpace(text) == "" {
		w.fail(lineOf(entry.Value), path, "is required")
	}
	return text
}

func (w *walker) chats(entry *ast.MappingValueNode, path string, parentLine int, depth domain.CatchUp) []domain.RequestChat {
	if entry == nil || isNull(entry.Value) {
		w.fail(parentLine, path, "a request needs at least one chat")
		return nil
	}
	list, ok := entry.Value.(*ast.SequenceNode)
	if !ok {
		w.fail(lineOf(entry.Value), path, "must be a list")
		return nil
	}
	if len(list.Values) == 0 {
		w.fail(lineOf(entry.Key), path, "a request needs at least one chat")
		return nil
	}
	chats := make([]domain.RequestChat, 0, len(list.Values))
	for i, item := range list.Values {
		chats = append(chats, w.chat(item, fmt.Sprintf("%s[%d]", path, i), depth))
	}
	return chats
}

func (w *walker) chat(node ast.Node, path string, requestDepth domain.CatchUp) domain.RequestChat {
	chat := domain.RequestChat{CatchUp: requestDepth}
	refNode := node
	if _, isMapping := entries(node); isMapping {
		found, _ := w.fields(node, path, "ref", "catch_up")
		if catchUp := found["catch_up"]; catchUp != nil {
			chat.CatchUp = w.depth(catchUp.Value, join(path, "catch_up")).over(requestDepth)
		}
		ref := found["ref"]
		if ref == nil || isNull(ref.Value) {
			w.fail(lineOf(node), join(path, "ref"), "is required")
			return chat
		}
		refNode, path = ref.Value, join(path, "ref")
	}
	text, ok := w.reference(refNode, path)
	if !ok {
		return chat
	}
	ref, err := domain.NewChatRef(text)
	if err != nil {
		w.fail(lineOf(refNode), path, err.Error())
	}
	chat.Ref = ref
	return chat
}
