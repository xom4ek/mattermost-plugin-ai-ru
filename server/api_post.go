package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/mattermost/mattermost-plugin-ai/server/ai"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

func (p *Plugin) postAuthorizationRequired(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	postID := c.Param("postid")

	post, err := p.pluginAPI.Post.GetPost(postID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Set(ContextPostKey, post)

	channel, err := p.pluginAPI.Channel.Get(post.ChannelId)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Set(ContextChannelKey, channel)

	if !p.pluginAPI.User.HasPermissionToChannel(userID, channel.Id, model.PermissionReadChannel) {
		c.AbortWithError(http.StatusForbidden, errors.New("user doesn't have permission to read channel post in in"))
		return
	}

	if err := p.checkUsageRestrictions(userID, channel); err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}
}

func (p *Plugin) handlePositivePostFeedback(c *gin.Context) {
	p.handlePostFeedback(c, true)
}
func (p *Plugin) handleNegativePostFeedback(c *gin.Context) {
	p.handlePostFeedback(c, false)
}

func (p *Plugin) handlePostFeedback(c *gin.Context, positive bool) {
	userID := c.GetHeader("Mattermost-User-Id")
	post := c.MustGet(ContextPostKey).(*model.Post)

	if _, err := p.execBuilder(p.builder.
		Insert("LLM_Feedback").
		SetMap(map[string]interface{}{
			"PostID":           post.Id,
			"UserID":           userID,
			"PositiveFeedback": positive,
		}).
		Suffix("ON CONFLICT (PostID) DO UPDATE SET PositiveFeedback = ?", positive)); err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "couldn't insert feedback"))
		return
	}

	c.Status(http.StatusOK)
}

func (p *Plugin) handleReact(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	post := c.MustGet(ContextPostKey).(*model.Post)
	channel := c.MustGet(ContextChannelKey).(*model.Channel)

	user, err := p.pluginAPI.User.Get(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err := p.selectEmoji(post, p.MakeConversationContext(user, channel, nil)); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

func (p *Plugin) handleSummarize(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	post := c.MustGet(ContextPostKey).(*model.Post)
	channel := c.MustGet(ContextChannelKey).(*model.Channel)

	user, err := p.pluginAPI.User.Get(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	createdPost, err := p.startNewSummaryThread(post.Id, p.MakeConversationContext(user, channel, nil))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "Unable to produce summary"))
		return
	}

	data := struct {
		PostID    string `json:"postid"`
		ChannelID string `json:"channelid"`
	}{
		PostID:    createdPost.Id,
		ChannelID: createdPost.ChannelId,
	}
	c.Render(http.StatusOK, render.JSON{Data: data})
}

func (p *Plugin) handleJiraTicket(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	post := c.MustGet(ContextPostKey).(*model.Post)
	channel := c.MustGet(ContextChannelKey).(*model.Channel)

	user, err := p.pluginAPI.User.Get(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	createdPost, err := p.startNewSummaryThread(post.Id, p.MakeConversationContext(user, channel, nil))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "Unable to create Jira ticket"))
		return
	}

	data := struct {
		PostID    string `json:"postid"`
		ChannelID string `json:"channelid"`
	}{
		PostID:    createdPost.Id,
		ChannelID: createdPost.ChannelId,
	}
	c.Render(http.StatusOK, render.JSON{Data: data})
}


func (p *Plugin) handleTranscribe(c *gin.Context) {
	post := c.MustGet(ContextPostKey).(*model.Post)
	channel := c.MustGet(ContextChannelKey).(*model.Channel)

	if err := p.handleCallRecordingPost(post, channel); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func (p *Plugin) handleStop(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	p.streamingContextsMutex.Lock()
	defer p.streamingContextsMutex.Unlock()

	post := c.MustGet(ContextPostKey).(*model.Post)

	if post.UserId != p.botid {
		c.AbortWithError(http.StatusBadRequest, errors.New("not a bot post"))
		return
	}

	if post.GetProp("llm_requester_user_id") != userID {
		c.AbortWithError(http.StatusForbidden, errors.New("only the original poster can stop the stream"))
		return
	}

	cancel, ok := p.streamingContexts[post.Id]
	if !ok {
		c.AbortWithError(http.StatusBadRequest, errors.New("context already canceled"))
		return
	}

	cancel()
}

func (p *Plugin) handleRegenerate(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	post := c.MustGet(ContextPostKey).(*model.Post)
	channel := c.MustGet(ContextChannelKey).(*model.Channel)

	user, err := p.pluginAPI.User.Get(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	threadData, err := p.getThreadAndMeta(post.RootId)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	threadData.cutoffAtPostID(post.Id)

	postToRegenerate := threadData.latestPost()

	if user.Id != postToRegenerate.UserId {
		c.AbortWithError(http.StatusForbidden, errors.New("only the original poster can regenerate"))
		return
	}

	context := p.MakeConversationContext(user, channel, postToRegenerate)
	conversation, err := p.prompts.ChatCompletion(ai.PromptDirectMessageQuestion, context)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	conversation.AppendConversation(ai.ThreadToBotConversation(p.botid, threadData.Posts))

	result, err := p.getLLM().ChatCompletion(conversation)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	post.Message = ""

	p.streamResultToPost(result, post)
}
