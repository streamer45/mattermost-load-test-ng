// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"errors"
	"io"
	"io/ioutil"

	"github.com/mattermost/mattermost-server/v5/model"
)

func (ue *UserEntity) SignUp(email, username, password string) error {
	user := model.User{
		Email:    email,
		Username: username,
		Password: password,
	}

	newUser, resp := ue.client.CreateUser(&user)
	if resp.Error != nil {
		return resp.Error
	}

	newUser.Password = password
	return ue.store.SetUser(newUser)
}

func (ue *UserEntity) Login() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	loggedUser, resp := ue.client.Login(user.Email, user.Password)
	if resp.Error != nil {
		return resp.Error
	}

	// We need to set user again because the user ID does not get set
	// if a user is already signed up.
	if err := ue.store.SetUser(loggedUser); err != nil {
		return err
	}

	return nil
}

func (ue *UserEntity) Logout() (bool, error) {
	ok, resp := ue.client.Logout()
	if resp.Error != nil {
		return false, resp.Error
	}

	return ok, nil
}

func (ue *UserEntity) GetConfig() error {
	config, resp := ue.client.GetConfig()
	if resp.Error != nil {
		return resp.Error
	}
	ue.store.SetConfig(config)
	return nil
}

func (ue *UserEntity) GetMe() (string, error) {
	user, resp := ue.client.GetMe("")
	if resp.Error != nil {
		return "", resp.Error
	}

	if err := ue.store.SetUser(user); err != nil {
		return "", err
	}

	return user.Id, nil
}

func (ue *UserEntity) GetPreferences() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	preferences, resp := ue.client.GetPreferences(user.Id)
	if resp.Error != nil {
		return resp.Error
	}

	if err := ue.store.SetPreferences(&preferences); err != nil {
		return err
	}
	return nil
}

func (ue *UserEntity) UpdatePreferences(pref *model.Preferences) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	if pref == nil {
		return errors.New("userentity: pref should not be nil")
	}

	ok, resp := ue.client.UpdatePreferences(user.Id, pref)
	if resp.Error != nil {
		return resp.Error
	} else if !ok {
		return errors.New("userentity: failed to update preferences")
	}

	return nil
}

func (ue *UserEntity) CreateUser(user *model.User) (string, error) {
	user, resp := ue.client.CreateUser(user)
	if resp.Error != nil {
		return "", resp.Error
	}

	return user.Id, nil
}

func (ue *UserEntity) UpdateUser(user *model.User) error {
	user, resp := ue.client.UpdateUser(user)
	if resp.Error != nil {
		return resp.Error
	}

	if user.Id == ue.store.Id() {
		return ue.store.SetUser(user)
	}

	return nil
}

func (ue *UserEntity) UpdateUserRoles(userId, roles string) error {
	_, resp := ue.client.UpdateUserRoles(userId, roles)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) PatchUser(userId string, patch *model.UserPatch) error {
	user, resp := ue.client.PatchUser(userId, patch)

	if resp.Error != nil {
		return resp.Error
	}

	if userId == ue.store.Id() {
		return ue.store.SetUser(user)
	}

	return nil
}

func (ue *UserEntity) CreatePost(post *model.Post) (string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	post.PendingPostId = model.NewId()
	post.UserId = user.Id

	post, resp := ue.client.CreatePost(post)
	if resp.Error != nil {
		return "", resp.Error
	}

	err = ue.store.SetPost(post)

	return post.Id, err
}

func (ue *UserEntity) PatchPost(postId string, patch *model.PostPatch) (string, error) {
	post, resp := ue.client.PatchPost(postId, patch)
	if resp.Error != nil {
		return "", resp.Error
	}

	if err := ue.store.SetPost(post); err != nil {
		return "", err
	}

	return post.Id, nil
}

func (ue *UserEntity) SearchPosts(teamId, terms string, isOrSearch bool) (*model.PostList, error) {
	postList, resp := ue.client.SearchPosts(teamId, terms, isOrSearch)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return postList, nil
}

func (ue *UserEntity) GetPostsForChannel(channelId string, page, perPage int) error {
	postList, resp := ue.client.GetPostsForChannel(channelId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil
	}
	return ue.store.SetPosts(postsMapToSlice(postList.Posts))
}

func (ue *UserEntity) GetPostsBefore(channelId, postId string, page, perPage int) error {
	postList, resp := ue.client.GetPostsBefore(channelId, postId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil
	}
	return ue.store.SetPosts(postsMapToSlice(postList.Posts))
}

func (ue *UserEntity) GetPostsAfter(channelId, postId string, page, perPage int) error {
	postList, resp := ue.client.GetPostsAfter(channelId, postId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil
	}
	return ue.store.SetPosts(postsMapToSlice(postList.Posts))
}

func (ue *UserEntity) GetPostsSince(channelId string, time int64) ([]string, error) {
	postList, resp := ue.client.GetPostsSince(channelId, time)
	if resp.Error != nil {
		return nil, resp.Error
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil, nil
	}

	return postList.Order, ue.store.SetPosts(postListToSlice(postList))
}

func (ue *UserEntity) GetPinnedPosts(channelId string) (*model.PostList, error) {
	postList, resp := ue.client.GetPinnedPosts(channelId, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	return postList, nil
}

// GetPostsAroundLastUnread returns the list of posts around last unread post by the current user in a channel.
func (ue *UserEntity) GetPostsAroundLastUnread(channelId string, limitBefore, limitAfter int) ([]string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	postList, resp := ue.client.GetPostsAroundLastUnread(user.Id, channelId, limitBefore, limitAfter)
	if resp.Error != nil {
		return nil, resp.Error
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil, nil
	}

	return postList.Order, ue.store.SetPosts(postListToSlice(postList))
}

func (ue *UserEntity) UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error) {
	fresp, resp := ue.client.UploadFile(data, channelId, filename)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return fresp, nil
}

func (ue *UserEntity) CreateChannel(channel *model.Channel) (string, error) {
	_, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	channel, resp := ue.client.CreateChannel(channel)
	if resp.Error != nil {
		return "", resp.Error
	}

	err = ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}

func (ue *UserEntity) CreateGroupChannel(memberIds []string) (string, error) {
	channel, resp := ue.client.CreateGroupChannel(memberIds)
	if resp.Error != nil {
		return "", resp.Error
	}

	err := ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}

func (ue *UserEntity) CreateDirectChannel(otherUserId string) (string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	channel, resp := ue.client.CreateDirectChannel(user.Id, otherUserId)
	if resp.Error != nil {
		return "", resp.Error
	}

	err = ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}
func (ue *UserEntity) RemoveUserFromChannel(channelId, userId string) (bool, error) {
	ok, resp := ue.client.RemoveUserFromChannel(channelId, userId)
	if resp.Error != nil {
		return false, resp.Error
	}
	return ok, ue.store.RemoveChannelMember(channelId, userId)
}

func (ue *UserEntity) AddChannelMember(channelId, userId string) error {
	member, resp := ue.client.AddChannelMember(channelId, userId)
	if resp.Error != nil {
		return nil
	}

	return ue.store.SetChannelMember(channelId, member)
}

func (ue *UserEntity) GetChannel(channelId string) error {
	channel, resp := ue.client.GetChannel(channelId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannel(channel)
}

func (ue *UserEntity) GetChannelsForTeam(teamId string, includeDeleted bool) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	channels, resp := ue.client.GetChannelsForTeamForUser(teamId, user.Id, includeDeleted, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannels(channels)
}

func (ue *UserEntity) GetPublicChannelsForTeam(teamId string, page, perPage int) error {
	channels, resp := ue.client.GetPublicChannelsForTeam(teamId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetChannels(channels)
}

func (ue *UserEntity) SearchChannels(teamId string, search *model.ChannelSearch) ([]*model.Channel, error) {
	channels, resp := ue.client.SearchChannels(teamId, search)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return channels, nil
}

func (ue *UserEntity) GetChannelsForTeamForUser(teamId, userId string, includeDeleted bool) ([]*model.Channel, error) {
	channels, resp := ue.client.GetChannelsForTeamForUser(teamId, userId, includeDeleted, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := ue.store.SetChannels(channels); err != nil {
		return nil, err
	}

	return channels, nil
}

func (ue *UserEntity) ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	channelViewResponse, resp := ue.client.ViewChannel(user.Id, view)
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := ue.store.SetChannelView(view.ChannelId); err != nil {
		return nil, err
	}

	return channelViewResponse, nil
}

func (ue *UserEntity) GetChannelUnread(channelId string) (*model.ChannelUnread, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	channelUnreadResponse, resp := ue.client.GetChannelUnread(channelId, user.Id)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return channelUnreadResponse, nil
}

func (ue *UserEntity) GetChannelMembers(channelId string, page, perPage int) error {
	channelMembers, resp := ue.client.GetChannelMembers(channelId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannelMembers(channelMembers)
}

// GetChannelMembersForUser gets all the channel members for a user on a team.
func (ue *UserEntity) GetChannelMembersForUser(userId, teamId string) error {
	channelMembers, resp := ue.client.GetChannelMembersForUser(userId, teamId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannelMembers(channelMembers)
}

func (ue *UserEntity) GetChannelMember(channelId, userId string) error {
	cm, resp := ue.client.GetChannelMember(channelId, userId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannelMember(channelId, cm)
}

func (ue *UserEntity) GetChannelStats(channelId string) error {
	_, resp := ue.client.GetChannelStats(channelId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

// AutocompleteChannelsForTeam returns an ordered list of channels for a given name.
func (ue *UserEntity) AutocompleteChannelsForTeam(teamId, name string) error {
	channelList, resp := ue.client.AutocompleteChannelsForTeam(teamId, name)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannels(*channelList)
}

func (ue *UserEntity) CreateTeam(team *model.Team) (string, error) {
	team, resp := ue.client.CreateTeam(team)
	if resp.Error != nil {
		return "", resp.Error
	}

	return team.Id, nil
}

func (ue *UserEntity) GetTeam(teamId string) error {
	team, resp := ue.client.GetTeam(teamId, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetTeam(team)
}

func (ue *UserEntity) UpdateTeam(team *model.Team) error {
	team, resp := ue.client.UpdateTeam(team)
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetTeam(team)
}

func (ue *UserEntity) GetTeamsForUser(userId string) ([]string, error) {
	teams, resp := ue.client.GetTeamsForUser(userId, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := ue.store.SetTeams(teams); err != nil {
		return nil, err
	}

	teamIds := make([]string, len(teams))
	for i, team := range teams {
		teamIds[i] = team.Id
	}

	return teamIds, nil
}

func (ue *UserEntity) AddTeamMember(teamId, userId string) error {
	tm, resp := ue.client.AddTeamMember(teamId, userId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetTeamMember(teamId, tm)
}

func (ue *UserEntity) RemoveTeamMember(teamId, userId string) error {
	_, resp := ue.client.RemoveTeamMember(teamId, userId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.RemoveTeamMember(teamId, userId)
}

func (ue *UserEntity) GetTeamMembers(teamId string, page, perPage int) error {
	members, resp := ue.client.GetTeamMembers(teamId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetTeamMembers(teamId, members)
}

func (ue *UserEntity) GetTeamMembersForUser(userId string) error {
	members, resp := ue.client.GetTeamMembersForUser(userId, "")
	if resp.Error != nil {
		return resp.Error
	}

	for _, m := range members {
		err := ue.store.SetTeamMember(m.TeamId, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ue *UserEntity) GetUsersByIds(userIds []string) ([]string, error) {
	users, resp := ue.client.GetUsersByIds(userIds)
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := ue.store.SetUsers(users); err != nil {
		return nil, err
	}

	newUserIds := make([]string, len(users))
	for i, user := range users {
		newUserIds[i] = user.Id
	}
	return newUserIds, nil
}

func (ue *UserEntity) GetUsersByUsernames(usernames []string) ([]string, error) {
	users, resp := ue.client.GetUsersByUsernames(usernames)
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := ue.store.SetUsers(users); err != nil {
		return nil, err
	}

	newUserIds := make([]string, len(users))
	for i, user := range users {
		newUserIds[i] = user.Id
	}
	return newUserIds, nil
}

func (ue *UserEntity) GetUserStatus() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	_, resp := ue.client.GetUserStatus(user.Id, "")
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) GetUsersStatusesByIds(userIds []string) error {
	statusList, resp := ue.client.GetUsersStatusesByIds(userIds)
	if resp.Error != nil {
		return resp.Error
	}

	for _, status := range statusList {
		if err := ue.store.SetStatus(status.UserId, status); err != nil {
			return err
		}
	}

	return nil
}

func (ue *UserEntity) GetUsersInChannel(channelId string, page, perPage int) error {
	if len(channelId) == 0 {
		return errors.New("userentity: channelId should not be empty")
	}

	users, resp := ue.client.GetUsersInChannel(channelId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetUsers(users)
}

func (ue *UserEntity) GetUsers(page, perPage int) ([]string, error) {
	users, resp := ue.client.GetUsers(page, perPage, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	userIds := make([]string, len(users))
	for i := range users {
		userIds[i] = users[i].Id
	}

	return userIds, ue.store.SetUsers(users)
}

func (ue *UserEntity) GetTeamStats(teamId string) error {
	_, resp := ue.client.GetTeamStats(teamId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) GetTeamsUnread(teamIdToExclude string) ([]*model.TeamUnread, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	unread, resp := ue.client.GetTeamsUnreadForUser(user.Id, teamIdToExclude)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return unread, nil
}

func (ue *UserEntity) GetFileInfosForPost(postId string) ([]*model.FileInfo, error) {
	infos, resp := ue.client.GetFileInfosForPost(postId, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	return infos, nil
}

func (ue *UserEntity) GetFileThumbnail(fileId string) error {
	_, resp := ue.client.GetFileThumbnail(fileId)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (ue *UserEntity) GetFilePreview(fileId string) error {
	_, resp := ue.client.GetFilePreview(fileId)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) AddTeamMemberFromInvite(token, inviteId string) error {
	tm, resp := ue.client.AddTeamMemberFromInvite(token, inviteId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetTeamMember(tm.TeamId, tm)
}

func (ue *UserEntity) SetProfileImage(data []byte) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	ok, resp := ue.client.SetProfileImage(user.Id, data)
	if resp.Error != nil {
		return resp.Error
	}
	if !ok {
		return errors.New("cannot set profile image")
	}
	return nil
}

func (ue *UserEntity) GetProfileImage() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	return ue.GetProfileImageForUser(user.Id)
}

func (ue *UserEntity) GetProfileImageForUser(userId string) error {
	if _, resp := ue.client.GetProfileImage(userId, ""); resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetProfileImage(userId)
}

func (ue *UserEntity) SearchUsers(search *model.UserSearch) ([]*model.User, error) {
	users, resp := ue.client.SearchUsers(search)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return users, nil
}

func (ue *UserEntity) AutoCompleteUsersInChannel(teamId, channelId, username string, limit int) (map[string]bool, error) {
	users, resp := ue.client.AutocompleteUsersInChannel(teamId, channelId, username, limit, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	if users == nil {
		return nil, errors.New("nil users")
	}
	usersMap := make(map[string]bool, len(users.Users)+len(users.OutOfChannel))
	for _, u := range users.Users {
		usersMap[u.Username] = true
	}
	for _, u := range users.OutOfChannel {
		usersMap[u.Username] = false
	}

	return usersMap, nil
}

func (ue *UserEntity) GetEmojiList(page, perPage int) error {
	emojis, resp := ue.client.GetEmojiList(page, perPage)
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetEmojis(emojis)
}

func (ue *UserEntity) GetEmojiImage(emojiId string) error {
	_, resp := ue.client.GetEmojiImage(emojiId)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) GetReactions(postId string) error {
	reactions, resp := ue.client.GetReactions(postId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetReactions(postId, reactions)
}

func (ue *UserEntity) SaveReaction(reaction *model.Reaction) error {
	r, resp := ue.client.SaveReaction(reaction)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetReaction(r)
}

func (ue *UserEntity) DeleteReaction(reaction *model.Reaction) error {
	_, resp := ue.client.DeleteReaction(reaction)
	if resp.Error != nil {
		return resp.Error
	}

	if _, err := ue.store.DeleteReaction(reaction); err != nil {
		return err
	}

	return nil
}

func (ue *UserEntity) GetTeams() ([]string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	return ue.GetTeamsForUser(user.Id)
}

// GetAllTeams returns all teams based on permissions.
func (ue *UserEntity) GetAllTeams(page, perPage int) ([]string, error) {
	teams, resp := ue.client.GetAllTeams("", page, perPage)
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := ue.store.SetTeams(teams); err != nil {
		return nil, err
	}

	teamIds := make([]string, len(teams))
	for i, team := range teams {
		teamIds[i] = team.Id
	}

	return teamIds, nil
}

func (ue *UserEntity) GetRolesByNames(roleNames []string) ([]string, error) {
	roles, resp := ue.client.GetRolesByNames(roleNames)
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := ue.store.SetRoles(roles); err != nil {
		return nil, err
	}

	roleIds := make([]string, len(roles))
	for i, role := range roles {
		roleIds[i] = role.Id
	}
	return roleIds, nil
}

func (ue *UserEntity) GetWebappPlugins() error {
	_, resp := ue.client.GetWebappPlugins()
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

// GetClientLicense returns the client license in the old format.
func (ue *UserEntity) GetClientLicense() error {
	license, resp := ue.client.GetOldClientLicense("")
	if resp.Error != nil {
		return resp.Error
	}
	if err := ue.store.SetLicense(license); err != nil {
		return err
	}
	return nil
}

func (ue *UserEntity) SetCurrentTeam(team *model.Team) error {
	return ue.store.SetCurrentTeam(team)
}

func (ue *UserEntity) SetCurrentChannel(channel *model.Channel) error {
	return ue.store.SetCurrentChannel(channel)
}

func (ue *UserEntity) ClearUserData() {
	ue.store.Clear()
}

// GetLogs fetches the logs.
func (ue *UserEntity) GetLogs(page, perPage int) error {
	_, resp := ue.client.GetLogs(page, perPage)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

// GetAnalytics fetches the system analytics.
func (ue *UserEntity) GetAnalytics() error {
	_, resp := ue.client.GetAnalyticsOld("", "")
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

// GetClusterStatus fetches the cluster status.
func (ue *UserEntity) GetClusterStatus() error {
	_, resp := ue.client.GetClusterStatus()
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

// GetPluginStatuses fetches the plugin statuses.
func (ue *UserEntity) GetPluginStatuses() error {
	// Need to do it manually until MM-25405 is resolved.
	_, resp := ue.getPluginStatuses()
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

// UpdateConfig updates the config with cfg.
func (ue *UserEntity) UpdateConfig(cfg *model.Config) error {
	cfg, resp := ue.client.UpdateConfig(cfg)
	if resp.Error != nil {
		return resp.Error
	}
	ue.store.SetConfig(cfg)
	return nil
}

func (ue *UserEntity) getPluginStatuses() (model.PluginStatuses, *model.Response) {
	r, err := ue.client.DoApiGet(ue.client.GetPluginsRoute()+"/statuses", "")
	if err != nil {
		return nil, model.BuildErrorResponse(r, err)
	}
	defer func() {
		_, _ = io.Copy(ioutil.Discard, r.Body)
		_ = r.Body.Close()
	}()
	return model.PluginStatusesFromJson(r.Body), model.BuildResponse(r)
}
