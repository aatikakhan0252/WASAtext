<template>
	<div class="d-flex mb-2" :class="isMine ? 'justify-content-end' : 'justify-content-start'">
		<div
			class="p-2 rounded shadow-sm position-relative"
			:class="isMine ? 'bg-success text-white' : 'bg-white'"
			style="max-width: 65%; min-width: 140px;"
		>
			<!-- Forwarded label -->
			<div v-if="message.isForwarded" class="small fst-italic mb-1" :class="isMine ? 'text-white-50' : 'text-muted'">
				↪ Forwarded
			</div>

			<!-- Sender name (not mine, for groups) -->
			<div v-if="!isMine" class="fw-bold small text-primary mb-1">{{ message.senderName }}</div>

			<!-- Reply snippet -->
			<div v-if="message.replyTo" class="border-start border-3 border-primary ps-2 mb-1 small rounded"
				 :class="isMine ? 'bg-white bg-opacity-10' : 'bg-light'">
				<div :class="isMine ? 'text-white-50' : 'text-muted'">
					{{ message.replyToContent || '' }}
					<span v-if="message.replyToHasPhoto">📷 Photo</span>
					<span v-if="!message.replyToContent && !message.replyToHasPhoto">Original message</span>
				</div>
			</div>

			<!-- Photo -->
			<div v-if="message.hasPhoto" class="mb-1">
				<img :src="photoUrl" alt="Photo" class="img-fluid rounded" style="max-height: 250px; cursor: pointer;" @error="imgError = true"/>
				<div v-if="imgError" class="text-muted">📷 Photo</div>
			</div>

			<!-- Text content -->
			<div v-if="message.content">{{ message.content }}</div>

			<!-- Timestamp + checkmarks -->
			<div class="d-flex justify-content-between align-items-center mt-1">
				<small :class="isMine ? 'text-white-50' : 'text-muted'">
					{{ formatTime(message.timestamp) }}
					<!-- Checkmarks: only on MY messages -->
					<template v-if="isMine">
						<span v-if="message.status === 'read'" style="color: #53bdeb;">✓✓</span>
						<span v-else-if="message.status === 'received'">✓✓</span>
						<span v-else>✓</span>
					</template>
				</small>
				<!-- Action buttons -->
				<div class="ms-2">
					<button @click="$emit('reply')" class="btn btn-link btn-sm p-0 ms-1" :class="isMine ? 'text-white' : ''" title="Reply">↩️</button>
					<button @click="$emit('forward')" class="btn btn-link btn-sm p-0 ms-1" :class="isMine ? 'text-white' : ''" title="Forward">↗️</button>
					<button @click="$emit('react')" class="btn btn-link btn-sm p-0 ms-1" :class="isMine ? 'text-white' : ''" title="React">😀</button>
					<button v-if="isMine" @click="$emit('delete')" class="btn btn-link btn-sm p-0 ms-1 text-danger" title="Delete">🗑️</button>
				</div>
			</div>

			<!-- Comments/Reactions with author names visible -->
			<div v-if="message.comments && message.comments.length > 0" class="mt-1 pt-1 border-top">
				<span v-for="comment in message.comments" :key="comment.userId"
					  class="badge me-1"
					  :class="comment.userId === currentUserId ? 'bg-primary' : 'bg-light text-dark'"
					  :title="comment.userName"
					  role="button"
					  @click="comment.userId === currentUserId ? $emit('remove-react') : null">
					{{ comment.emoticon }} <small>{{ comment.userName }}</small>
				</span>
			</div>
		</div>
	</div>
</template>

<script>
import api from '@/services/api.js';

export default {
	name: 'MessageBubble',
	props: {
		message: {type: Object, required: true},
		isMine: {type: Boolean, default: false},
		conversationId: {type: String, default: ''},
	},
	emits: ['delete', 'react', 'remove-react', 'reply', 'forward'],
	data() {
		return {
			imgError: false,
		};
	},
	computed: {
		currentUserId() {
			return sessionStorage.getItem('userId');
		},
		photoUrl() {
			if (this.message.hasPhoto && this.conversationId) {
				return api.getMessagePhotoUrl(this.conversationId, this.message.messageId);
			}
			return '';
		},
	},
	methods: {
		formatTime(timestamp) {
			if (!timestamp) return '';
			const date = new Date(timestamp);
			return date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'});
		},
	},
};
</script>
