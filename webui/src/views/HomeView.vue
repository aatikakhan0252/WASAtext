<template>
	<div class="d-flex vh-100">
		<!-- SIDEBAR -->
		<div class="d-flex flex-column bg-white border-end" style="width: 350px; min-width: 350px;">
			<!-- Sidebar Header -->
			<div class="d-flex justify-content-between align-items-center p-3 bg-light border-bottom">
				<div class="d-flex align-items-center">
					<strong>{{ userName }}</strong>
				</div>
				<div>
					<button @click="showSearch = !showSearch" class="btn btn-sm btn-outline-success me-1" title="New Chat">➕</button>
					<button @click="logout" class="btn btn-sm btn-outline-secondary" title="Logout">🚪</button>
				</div>
			</div>

			<!-- User Search -->
			<div v-if="showSearch" class="p-2 bg-light">
				<div class="input-group input-group-sm">
					<input v-model="searchQuery" @input="searchUsers" class="form-control" placeholder="Search users..."/>
					<button @click="showSearch = false; searchQuery = ''; searchResults = []" class="btn btn-outline-secondary">✕</button>
				</div>
				<div class="list-group mt-1" style="max-height: 200px; overflow-y: auto;">
					<a v-for="u in searchResults" :key="u.identifier"
						 @click="startChat(u.identifier)" href="#"
						 class="list-group-item list-group-item-action py-2">
						{{ u.name }}
					</a>
				</div>
			</div>

			<!-- Conversation List -->
			<div class="flex-grow-1 overflow-auto">
				<div v-if="conversations.length === 0" class="text-center text-muted p-4">
					No conversations yet. Search for a user to start chatting!
				</div>
				<ConversationItem
					v-for="conv in conversations"
					:key="conv.conversationId"
					:conversation="conv"
					:active="activeConv && activeConv.conversationId === conv.conversationId"
					@select="selectConversation(conv)"
				/>
			</div>
		</div>

		<!-- CHAT AREA -->
		<div class="flex-grow-1 d-flex flex-column">
			<template v-if="activeConv">
				<!-- Chat Header -->
				<div class="d-flex align-items-center p-3 bg-light border-bottom">
					<strong>{{ activeConv.name }}</strong>
					<span v-if="activeConv.isGroup" class="badge bg-secondary ms-2">Group</span>
				</div>

				<!-- Messages -->
				<div ref="msgList" class="flex-grow-1 overflow-auto p-3" style="background: #e5ddd5;">
					<MessageBubble
						v-for="msg in messages"
						:key="msg.messageId"
						:message="msg"
						:is-mine="msg.senderId === userId"
						@delete="deleteMsg(msg)"
						@react="reactToMsg(msg)"
					/>
				</div>

				<!-- Input -->
				<div class="d-flex p-3 bg-light border-top">
					<input
						v-model="newMessage"
						@keyup.enter="sendMessage"
						type="text"
						class="form-control me-2"
						placeholder="Type a message"
					/>
					<button @click="sendMessage" class="btn btn-success" :disabled="!newMessage">Send</button>
				</div>
			</template>
			<div v-else class="flex-grow-1 d-flex justify-content-center align-items-center text-muted">
				<p>Select a conversation to start messaging</p>
			</div>
		</div>
	</div>
</template>

<script>
import api from '@/services/api.js';
import ConversationItem from '@/components/ConversationItem.vue';
import MessageBubble from '@/components/MessageBubble.vue';

export default {
	name: 'HomeView',
	components: {ConversationItem, MessageBubble},
	data() {
		return {
			userId: sessionStorage.getItem('userId'),
			userName: sessionStorage.getItem('userName'),
			conversations: [],
			activeConv: null,
			messages: [],
			newMessage: '',
			showSearch: false,
			searchQuery: '',
			searchResults: [],
			pollInterval: null,
			errorMsg: null,
		};
	},
	mounted() {
		if (!this.userId) {
			this.$router.push('/login');
			return;
		}
		this.refreshConversations();
		this.pollInterval = setInterval(() => {
			this.refreshConversations();
			if (this.activeConv) {
				this.refreshMessages();
			}
		}, 3000);
	},
	beforeUnmount() {
		if (this.pollInterval) {
			clearInterval(this.pollInterval);
		}
	},
	methods: {
		async refreshConversations() {
			try {
				this.conversations = await api.getMyConversations() || [];
			} catch (e) {
				console.error('Error fetching conversations:', e);
			}
		},
		async selectConversation(conv) {
			this.activeConv = conv;
			await this.refreshMessages();
		},
		async refreshMessages() {
			if (!this.activeConv) return;
			try {
				const data = await api.getConversation(this.activeConv.conversationId);
				this.messages = data.messages || [];
				this.$nextTick(() => {
					const container = this.$refs.msgList;
					if (container) container.scrollTop = container.scrollHeight;
				});
			} catch (e) {
				console.error('Error fetching messages:', e);
			}
		},
		async sendMessage() {
			if (!this.newMessage || !this.activeConv) return;
			const content = this.newMessage;
			this.newMessage = '';
			try {
				await api.sendMessage(this.activeConv.conversationId, content);
				await this.refreshMessages();
				await this.refreshConversations();
			} catch (e) {
				console.error('Error sending message:', e);
				this.newMessage = content;
			}
		},
		async searchUsers() {
			if (!this.searchQuery) {
				this.searchResults = [];
				return;
			}
			try {
				const results = await api.searchUsers(this.searchQuery) || [];
				this.searchResults = results.filter(u => u.identifier !== this.userId);
			} catch (e) {
				console.error('Error searching users:', e);
			}
		},
		async startChat(targetId) {
			try {
				const data = await api.startConversation(targetId);
				this.showSearch = false;
				this.searchQuery = '';
				this.searchResults = [];
				await this.refreshConversations();
				const newConv = this.conversations.find(c => c.conversationId === data.conversationId);
				if (newConv) this.selectConversation(newConv);
			} catch (e) {
				console.error('Error starting conversation:', e);
			}
		},
		async deleteMsg(msg) {
			if (!this.activeConv) return;
			try {
				await api.deleteMessage(this.activeConv.conversationId, msg.messageId);
				await this.refreshMessages();
			} catch (e) {
				console.error('Error deleting message:', e);
			}
		},
		async reactToMsg(msg) {
			if (!this.activeConv) return;
			try {
				await api.commentMessage(this.activeConv.conversationId, msg.messageId, '👍');
				await this.refreshMessages();
			} catch (e) {
				console.error('Error reacting to message:', e);
			}
		},
		logout() {
			sessionStorage.removeItem('userId');
			sessionStorage.removeItem('userName');
			this.$router.push('/login');
		},
	},
};
</script>
