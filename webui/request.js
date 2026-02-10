/*
API Helper Functions
Uses axios for HTTP requests.
*/

const API_BASE = "http://localhost:3000";

const api = {
    // LOGIN
    async login(username) {
        try {
            const response = await axios.post(`${API_BASE}/session`, { name: username });
            return response.data; // { identifier: "..." }
        } catch (e) {
            throw e.response ? e.response.data : e;
        }
    },

    // GET CONVERSATIONS
    async getConversations(userId) {
        try {
            const response = await axios.get(`${API_BASE}/conversations`, {
                headers: { 'Authorization': `Bearer ${userId}` }
            });
            return response.data;
        } catch (e) {
            console.error("Error fetching conversations:", e);
            throw e;
        }
    },

    // GET MESSAGES
    async getConversation(userId, conversationId) {
        try {
            const response = await axios.get(`${API_BASE}/conversations/${conversationId}`, {
                headers: { 'Authorization': `Bearer ${userId}` }
            });
            return response.data;
        } catch (e) {
            console.error("Error fetching messages:", e);
            throw e;
        }
    },

    // SEND MESSAGE
    async sendMessage(userId, conversationId, content) {
        try {
            const response = await axios.post(`${API_BASE}/conversations/${conversationId}/messages`,
                { content: content },
                { headers: { 'Authorization': `Bearer ${userId}` } }
            );
            return response.data;
        } catch (e) {
            console.error("Error sending message:", e);
            throw e;
        }
    },

    // START CONVERSATION
    async startConversation(userId, targetUserId) {
        try {
            const response = await axios.post(`${API_BASE}/conversations`,
                { userId: targetUserId },
                { headers: { 'Authorization': `Bearer ${userId}` } }
            );
            return response.data; // { conversationId: "..." }
        } catch (e) {
            console.error("Error starting conversation:", e);
            throw e;
        }
    },

    // SEARCH USERS
    async searchUsers(userId, query) {
        try {
            const response = await axios.get(`${API_BASE}/users?search=${query}`, {
                headers: { 'Authorization': `Bearer ${userId}` }
            });
            return response.data;
        } catch (e) {
            console.error("Error searching users:", e);
            throw e;
        }
    }
};
