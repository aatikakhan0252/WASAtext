import axios from 'axios';

const instance = axios.create({
    baseURL: __API_URL__,
    timeout: 10000,
});

// Interceptor to add Authorization header
instance.interceptors.request.use((config) => {
    const userId = sessionStorage.getItem('userId');
    if (userId) {
        config.headers['Authorization'] = `Bearer ${userId}`;
    }
    return config;
});

export default {
    // LOGIN
    async login(username) {
        const response = await instance.post('/session', { name: username });
        return response.data;
    },

    // USERS
    async searchUsers(query) {
        const response = await instance.get('/users', { params: { search: query } });
        return response.data;
    },
    async setMyUserName(userId, name) {
        const response = await instance.put(`/users/${userId}/username`, { name: name });
        return response.data;
    },
    async setMyPhoto(userId, photoFile) {
        const response = await instance.put(`/users/${userId}/photo`, photoFile, {
            headers: { 'Content-Type': 'application/octet-stream' }
        });
        return response.data;
    },
    getUserPhotoUrl(userId) {
        return `${__API_URL__}/users/${userId}/photo?t=${Date.now()}`;
    },

    // CONVERSATIONS
    async getMyConversations() {
        const response = await instance.get('/conversations');
        return response.data;
    },
    async getConversation(conversationId) {
        const response = await instance.get(`/conversations/${conversationId}`);
        return response.data;
    },
    async startConversation(targetUserId) {
        const response = await instance.post('/conversations', { userId: targetUserId });
        return response.data;
    },

    // MESSAGES
    async sendMessage(conversationId, content, replyTo) {
        const body = { content: content };
        if (replyTo) body.replyTo = replyTo;
        const response = await instance.post(`/conversations/${conversationId}/messages`, body);
        return response.data;
    },
    async sendMessageWithPhoto(conversationId, content, photoFile, replyTo) {
        const formData = new FormData();
        if (content) formData.append('content', content);
        if (photoFile) formData.append('photo', photoFile);
        if (replyTo) formData.append('replyTo', replyTo);
        const response = await instance.post(`/conversations/${conversationId}/messages`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
        return response.data;
    },
    getMessagePhotoUrl(conversationId, messageId) {
        return `${__API_URL__}/conversations/${conversationId}/messages/${messageId}/photo`;
    },
    async deleteMessage(conversationId, messageId) {
        const response = await instance.delete(`/conversations/${conversationId}/messages/${messageId}`);
        return response.data;
    },
    async forwardMessage(conversationId, messageId, targetConversationId) {
        const response = await instance.post(`/conversations/${conversationId}/messages/${messageId}/forward`, {
            targetConversationId: targetConversationId
        });
        return response.data;
    },

    // COMMENTS (REACTIONS)
    async commentMessage(conversationId, messageId, emoticon) {
        const response = await instance.post(`/conversations/${conversationId}/messages/${messageId}/comments`, {
            emoticon: emoticon
        });
        return response.data;
    },
    async uncommentMessage(conversationId, messageId) {
        const response = await instance.delete(`/conversations/${conversationId}/messages/${messageId}/comments`);
        return response.data;
    },

    // GROUPS
    async createGroup(name, memberIds) {
        const response = await instance.post('/groups', { name: name, memberIds: memberIds });
        return response.data;
    },
    async addToGroup(groupId, userId) {
        const response = await instance.post(`/groups/${groupId}/members`, { userId: userId });
        return response.data;
    },
    async leaveGroup(groupId) {
        const response = await instance.delete(`/groups/${groupId}/members/me`);
        return response.data;
    },
    async setGroupName(groupId, name) {
        const response = await instance.put(`/groups/${groupId}/name`, { name: name });
        return response.data;
    },
    async setGroupPhoto(groupId, photoFile) {
        const response = await instance.put(`/groups/${groupId}/photo`, photoFile, {
            headers: { 'Content-Type': 'application/octet-stream' }
        });
        return response.data;
    },
    getGroupPhotoUrl(groupId) {
        return `${__API_URL__}/groups/${groupId}/photo?t=${Date.now()}`;
    },
};
