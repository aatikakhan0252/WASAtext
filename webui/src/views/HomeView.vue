<template>
	<div class="d-flex vh-100">
		<!-- SIDEBAR -->
		<div class="d-flex flex-column bg-white border-end" style="width: 350px; min-width: 350px;">
			<!-- Sidebar Header -->
			<div class="d-flex justify-content-between align-items-center p-3 bg-light border-bottom">
				<div class="d-flex align-items-center" role="button" @click="showProfileSettings = true">
					<strong>{{ userName }}</strong>
					<small class="ms-1 text-muted">⚙️</small>
				</div>
				<div>
					<button @click="showSearch = !showSearch" class="btn btn-sm btn-outline-success me-1" title="New Chat">💬</button>
					<button @click="showGroupCreate = true" class="btn btn-sm btn-outline-primary me-1" title="New Group">👥</button>
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
				<div class="d-flex align-items-center justify-content-between p-3 bg-light border-bottom">
					<div>
						<strong>{{ chatName }}</strong>
						<span v-if="activeConv.isGroup" class="badge bg-secondary ms-2">Group</span>
					</div>
					<div v-if="activeConv.isGroup">
						<button @click="showGroupSettings = true" class="btn btn-sm btn-outline-primary me-1" title="Group Settings">⚙️</button>
						<button @click="doLeaveGroup" class="btn btn-sm btn-outline-danger" title="Leave Group">🚪 Leave</button>
					</div>
				</div>

				<!-- Messages -->
				<div ref="msgList" class="flex-grow-1 overflow-auto p-3" style="background: #e5ddd5;">
					<MessageBubble
						v-for="msg in messages"
						:key="msg.messageId"
						:message="msg"
						:is-mine="msg.senderId === userId"
						:conversation-id="activeConv.conversationId"
						@delete="deleteMsg(msg)"
						@react="showEmojiPicker(msg)"
						@remove-react="removeReaction(msg)"
						@reply="setReplyTo(msg)"
						@forward="showForwardPicker(msg)"
					/>
				</div>

				<!-- Reply Preview -->
				<div v-if="replyToMsg" class="d-flex align-items-center px-3 py-2 bg-white border-top">
					<div class="flex-grow-1 small">
						<span class="text-primary fw-bold">Replying to {{ replyToMsg.senderName }}</span>
						<div class="text-muted text-truncate">{{ replyToMsg.content || '📷 Photo' }}</div>
					</div>
					<button @click="replyToMsg = null" class="btn btn-sm btn-link text-danger">✕</button>
				</div>

				<!-- Input -->
				<div class="d-flex align-items-center p-3 bg-light border-top">
					<label class="btn btn-sm btn-outline-secondary me-2 mb-0" title="Attach Photo">
						📷
						<input type="file" accept="image/*" @change="onPhotoSelect" class="d-none" ref="photoInput"/>
					</label>
					<div v-if="selectedPhoto" class="me-2">
						<span class="badge bg-info">{{ selectedPhoto.name }}</span>
						<button @click="selectedPhoto = null" class="btn btn-sm btn-link p-0 text-danger">✕</button>
					</div>
					<input
						v-model="newMessage"
						@keyup.enter="sendMessage"
						type="text"
						class="form-control me-2"
						placeholder="Type a message"
					/>
					<button @click="sendMessage" class="btn btn-success" :disabled="!newMessage && !selectedPhoto">Send</button>
				</div>
			</template>
			<div v-else class="flex-grow-1 d-flex justify-content-center align-items-center text-muted">
				<p>Select a conversation to start messaging</p>
			</div>
		</div>

		<!-- EMOJI PICKER MODAL -->
		<div v-if="emojiTarget" class="modal show d-block" style="background: rgba(0,0,0,0.3);" @click.self="emojiTarget = null">
			<div class="modal-dialog modal-sm modal-dialog-centered">
				<div class="modal-content p-3 text-center">
					<h6>Pick a reaction</h6>
					<div class="d-flex flex-wrap justify-content-center gap-2 fs-4">
						<span v-for="e in emojis" :key="e" role="button" @click="doReact(e)" class="p-1">{{ e }}</span>
					</div>
					<button @click="emojiTarget = null" class="btn btn-sm btn-secondary mt-2">Cancel</button>
				</div>
			</div>
		</div>

		<!-- FORWARD PICKER MODAL -->
		<div v-if="forwardTarget" class="modal show d-block" style="background: rgba(0,0,0,0.3);" @click.self="forwardTarget = null">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content p-3">
					<h5>Forward to...</h5>
					<div class="list-group" style="max-height: 300px; overflow-y: auto;">
						<a v-for="conv in conversations.filter(c => c.conversationId !== activeConv.conversationId)"
						   :key="conv.conversationId"
						   @click="doForward(conv.conversationId)" href="#"
						   class="list-group-item list-group-item-action">
							{{ conv.name }}
						</a>
					</div>
					<button @click="forwardTarget = null" class="btn btn-sm btn-secondary mt-2">Cancel</button>
				</div>
			</div>
		</div>

		<!-- GROUP CREATE MODAL -->
		<div v-if="showGroupCreate" class="modal show d-block" style="background: rgba(0,0,0,0.3);" @click.self="showGroupCreate = false">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content p-3">
					<h5>Create Group</h5>
					<input v-model="groupName" class="form-control mb-2" placeholder="Group name"/>
					<input v-model="groupSearchQuery" @input="searchGroupMembers" class="form-control mb-2" placeholder="Search users to add..."/>
					<div class="list-group mb-2" style="max-height: 150px; overflow-y: auto;">
						<a v-for="u in groupSearchResults" :key="u.identifier"
						   @click="toggleGroupMember(u)" href="#"
						   class="list-group-item list-group-item-action py-1"
						   :class="{'active': selectedGroupMembers.some(m => m.identifier === u.identifier)}">
							{{ u.name }}
						</a>
					</div>
					<div v-if="selectedGroupMembers.length" class="mb-2">
						<span v-for="m in selectedGroupMembers" :key="m.identifier" class="badge bg-primary me-1">
							{{ m.name }} <span @click="removeGroupMember(m)" role="button">✕</span>
						</span>
					</div>
					<div v-if="groupError" class="alert alert-danger py-1 mb-2">{{ groupError }}</div>
					<div class="d-flex gap-2">
						<button @click="doCreateGroup" class="btn btn-primary" :disabled="!groupName || selectedGroupMembers.length === 0">Create</button>
						<button @click="showGroupCreate = false" class="btn btn-secondary">Cancel</button>
					</div>
				</div>
			</div>
		</div>

		<!-- PROFILE SETTINGS MODAL -->
		<div v-if="showProfileSettings" class="modal show d-block" style="background: rgba(0,0,0,0.3);" @click.self="showProfileSettings = false">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content p-3">
					<h5>Profile Settings</h5>
					<div class="mb-3">
						<label class="form-label">Username</label>
						<div class="input-group">
							<input v-model="newUserName" class="form-control" placeholder="New username"/>
							<button @click="doChangeUserName" class="btn btn-primary" :disabled="!newUserName">Save</button>
						</div>
					</div>
					<div class="mb-3">
						<label class="form-label">Profile Photo</label>
						<input type="file" accept="image/*" @change="onProfilePhotoSelect" class="form-control"/>
					</div>
					<div v-if="profileError" class="alert alert-danger py-1 mb-2">{{ profileError }}</div>
					<div v-if="profileSuccess" class="alert alert-success py-1 mb-2">{{ profileSuccess }}</div>
					<button @click="showProfileSettings = false" class="btn btn-secondary">Close</button>
				</div>
			</div>
		</div>

		<!-- GROUP SETTINGS MODAL -->
		<div v-if="showGroupSettings && activeConv && activeConv.isGroup" class="modal show d-block" style="background: rgba(0,0,0,0.3);" @click.self="showGroupSettings = false">
			<div class="modal-dialog modal-dialog-centered">
				<div class="modal-content p-3">
					<h5>Group Settings</h5>
					<div class="mb-3">
						<label class="form-label">Group Name</label>
						<div class="input-group">
							<input v-model="newGroupName" class="form-control" placeholder="New group name"/>
							<button @click="doChangeGroupName" class="btn btn-primary" :disabled="!newGroupName">Save</button>
						</div>
					</div>
					<div class="mb-3">
						<label class="form-label">Group Photo</label>
						<input type="file" accept="image/*" @change="onGroupPhotoSelect" class="form-control"/>
					</div>
					<div v-if="groupSettingsMsg" class="alert alert-info py-1 mb-2">{{ groupSettingsMsg }}</div>
					<button @click="showGroupSettings = false" class="btn btn-secondary">Close</button>
				</div>
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
			chatName: '',
			messages: [],
			newMessage: '',
			selectedPhoto: null,
			replyToMsg: null,
			showSearch: false,
			searchQuery: '',
			searchResults: [],
			pollInterval: null,

			// Emoji picker
			emojiTarget: null,
			emojis: ['👍', '❤️', '😂', '😮', '😢', '🔥', '👏', '🎉'],

			// Forward
			forwardTarget: null,

			// Group creation
			showGroupCreate: false,
			groupName: '',
			groupSearchQuery: '',
			groupSearchResults: [],
			selectedGroupMembers: [],
			groupError: null,

			// Profile settings
			showProfileSettings: false,
			newUserName: '',
			profileError: null,
			profileSuccess: null,

			// Group settings
			showGroupSettings: false,
			newGroupName: '',
			groupSettingsMsg: null,
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
				// Update active conv name in case it changed (profile/group updates propagation)
				if (this.activeConv) {
					const updated = this.conversations.find(c => c.conversationId === this.activeConv.conversationId);
					if (updated) {
						this.activeConv = updated;
						this.chatName = updated.name;
					}
				}
			} catch (e) {
				console.error('Error fetching conversations:', e);
			}
		},
		async selectConversation(conv) {
			this.activeConv = conv;
			this.chatName = conv.name;
			this.replyToMsg = null;
			await this.refreshMessages();
		},
		async refreshMessages() {
			if (!this.activeConv) return;
			try {
				const data = await api.getConversation(this.activeConv.conversationId);
				this.messages = data.messages || [];
				// Update chat name from conversation data (propagates profile/group changes)
				this.chatName = data.name;
				if (data.groupId) {
					this.activeConv.groupId = data.groupId;
				}
				this.$nextTick(() => {
					const container = this.$refs.msgList;
					if (container) container.scrollTop = container.scrollHeight;
				});
			} catch (e) {
				if (e.response && e.response.status === 404) {
					// User left the group or conversation deleted
					this.activeConv = null;
					this.messages = [];
					this.refreshConversations();
				} else {
					console.error('Error fetching messages:', e);
				}
			}
		},
		async sendMessage() {
			if (!this.newMessage && !this.selectedPhoto) return;
			if (!this.activeConv) return;

			const content = this.newMessage;
			const photo = this.selectedPhoto;
			const replyTo = this.replyToMsg ? this.replyToMsg.messageId : null;

			this.newMessage = '';
			this.selectedPhoto = null;
			this.replyToMsg = null;

			try {
				if (photo) {
					await api.sendMessageWithPhoto(this.activeConv.conversationId, content, photo, replyTo);
				} else {
					await api.sendMessage(this.activeConv.conversationId, content, replyTo);
				}
				await this.refreshMessages();
				await this.refreshConversations();
			} catch (e) {
				console.error('Error sending message:', e);
				this.newMessage = content;
				this.selectedPhoto = photo;
			}
		},
		onPhotoSelect(event) {
			const file = event.target.files[0];
			if (file) {
				this.selectedPhoto = file;
			}
			if (this.$refs.photoInput) this.$refs.photoInput.value = '';
		},
		setReplyTo(msg) {
			this.replyToMsg = msg;
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

		// REACTIONS
		showEmojiPicker(msg) {
			this.emojiTarget = msg;
		},
		async doReact(emoji) {
			if (!this.emojiTarget || !this.activeConv) return;
			try {
				await api.commentMessage(this.activeConv.conversationId, this.emojiTarget.messageId, emoji);
				this.emojiTarget = null;
				await this.refreshMessages();
			} catch (e) {
				console.error('Error reacting:', e);
			}
		},
		async removeReaction(msg) {
			if (!this.activeConv) return;
			try {
				await api.uncommentMessage(this.activeConv.conversationId, msg.messageId);
				await this.refreshMessages();
			} catch (e) {
				console.error('Error removing reaction:', e);
			}
		},

		// FORWARDING
		showForwardPicker(msg) {
			this.forwardTarget = msg;
		},
		async doForward(targetConvId) {
			if (!this.forwardTarget || !this.activeConv) return;
			try {
				await api.forwardMessage(this.activeConv.conversationId, this.forwardTarget.messageId, targetConvId);
				this.forwardTarget = null;
				await this.refreshConversations();
			} catch (e) {
				console.error('Error forwarding:', e);
			}
		},

		// GROUP CREATION
		async searchGroupMembers() {
			if (!this.groupSearchQuery) {
				this.groupSearchResults = [];
				return;
			}
			try {
				const results = await api.searchUsers(this.groupSearchQuery) || [];
				this.groupSearchResults = results.filter(u => u.identifier !== this.userId);
			} catch (e) {
				console.error('Error searching users:', e);
			}
		},
		toggleGroupMember(user) {
			const idx = this.selectedGroupMembers.findIndex(m => m.identifier === user.identifier);
			if (idx >= 0) {
				this.selectedGroupMembers.splice(idx, 1);
			} else {
				this.selectedGroupMembers.push(user);
			}
		},
		removeGroupMember(member) {
			this.selectedGroupMembers = this.selectedGroupMembers.filter(m => m.identifier !== member.identifier);
		},
		async doCreateGroup() {
			this.groupError = null;
			try {
				const memberIds = this.selectedGroupMembers.map(m => m.identifier);
				await api.createGroup(this.groupName, memberIds);
				this.showGroupCreate = false;
				this.groupName = '';
				this.selectedGroupMembers = [];
				this.groupSearchQuery = '';
				this.groupSearchResults = [];
				await this.refreshConversations();
			} catch (e) {
				this.groupError = e.response?.data?.message || 'Failed to create group';
			}
		},

		// LEAVE GROUP
		async doLeaveGroup() {
			if (!this.activeConv || !this.activeConv.groupId) return;
			if (!confirm('Are you sure you want to leave this group?')) return;
			try {
				await api.leaveGroup(this.activeConv.groupId);
				this.activeConv = null;
				this.messages = [];
				await this.refreshConversations();
			} catch (e) {
				console.error('Error leaving group:', e);
			}
		},

		// PROFILE SETTINGS
		async doChangeUserName() {
			this.profileError = null;
			this.profileSuccess = null;
			try {
				await api.setMyUserName(this.userId, this.newUserName);
				this.userName = this.newUserName;
				sessionStorage.setItem('userName', this.newUserName);
				this.profileSuccess = 'Username updated!';
				this.newUserName = '';
				await this.refreshConversations();
			} catch (e) {
				this.profileError = e.response?.data?.message || 'Failed to update username. It may already be taken.';
			}
		},
		async onProfilePhotoSelect(event) {
			const file = event.target.files[0];
			if (!file) return;
			this.profileError = null;
			this.profileSuccess = null;
			try {
				const bytes = await file.arrayBuffer();
				await api.setMyPhoto(this.userId, bytes);
				this.profileSuccess = 'Photo updated!';
			} catch (e) {
				this.profileError = 'Failed to update photo';
			}
		},

		// GROUP SETTINGS
		async doChangeGroupName() {
			if (!this.activeConv || !this.activeConv.groupId) return;
			this.groupSettingsMsg = null;
			try {
				await api.setGroupName(this.activeConv.groupId, this.newGroupName);
				this.groupSettingsMsg = 'Group name updated!';
				this.newGroupName = '';
				await this.refreshConversations();
				await this.refreshMessages();
			} catch (e) {
				this.groupSettingsMsg = 'Failed to update name';
			}
		},
		async onGroupPhotoSelect(event) {
			if (!this.activeConv || !this.activeConv.groupId) return;
			const file = event.target.files[0];
			if (!file) return;
			this.groupSettingsMsg = null;
			try {
				const bytes = await file.arrayBuffer();
				await api.setGroupPhoto(this.activeConv.groupId, bytes);
				this.groupSettingsMsg = 'Group photo updated!';
				await this.refreshConversations();
			} catch (e) {
				this.groupSettingsMsg = 'Failed to update photo';
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
