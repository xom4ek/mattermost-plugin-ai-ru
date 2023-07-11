package ai

import (
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
)

type PostRole int

const (
	PostRoleUser PostRole = iota
	PostRoleBot
	PostRoleSystem
)

type Post struct {
	Role    PostRole
	Message string
}

type ConversationContext struct {
	RequestingUser   *model.User
	Channel          *model.Channel
	Post             *model.Post
	PromptParameters map[string]string
}

func NewConversationContext(reqeustingUser *model.User, channel *model.Channel, post *model.Post) ConversationContext {
	return ConversationContext{
		RequestingUser: reqeustingUser,
		Channel:        channel,
		Post:           post,
	}
}

func NewConversationContextParametersOnly(promptParameters map[string]string) ConversationContext {
	return ConversationContext{
		PromptParameters: promptParameters,
	}
}

type BotConversation struct {
	Posts   []Post
	Tools   []Tool
	Context ConversationContext
}

func (b *BotConversation) AddUserPost(post *model.Post) {
	b.Posts = append(b.Posts, Post{
		Role:    PostRoleUser,
		Message: post.Message,
	})
}

func (b *BotConversation) AppendConversation(conversation BotConversation) {
	b.Posts = append(b.Posts, conversation.Posts...)
}

func (b *BotConversation) ExtractSystemMessage() string {
	var result strings.Builder
	for _, post := range b.Posts {
		if post.Role == PostRoleSystem {
			result.WriteString(post.Message)
		}
	}
	return result.String()
}

func GetPostRole(botID string, post *model.Post) PostRole {
	if post.UserId == botID {
		return PostRoleBot
	}
	return PostRoleUser
}

func ThreadToBotConversation(botID string, posts []*model.Post) BotConversation {
	result := BotConversation{
		Posts: make([]Post, 0, len(posts)),
	}

	for _, post := range posts {
		result.Posts = append(result.Posts, Post{
			Role:    GetPostRole(botID, post),
			Message: post.Message,
		})
	}

	return result
}