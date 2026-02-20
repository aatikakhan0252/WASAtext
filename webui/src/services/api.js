import axios from 'axios';

const instance = axios.create({
    baseURL: location.origin,
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
    async setMyPhoto(userId, photoData) {
        const response = await instance.put(`/users/${userId}/photo`, photoData, {
            headers: { 'Content-Type': 'image/png' }
        });
        return response.data;
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
    async sendMessage(conversationId, content) {
        const response = await instance.post(`/conversations/${conversationId}/messages`, { content: content });
        return response.data;
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
    async setGroupPhoto(groupId, photoData) {
        const response = await instance.put(`/groups/${groupId}/photo`, photoData, {
            headers: { 'Content-Type': 'image/png' }
        });
        return response.data;
    },
};
