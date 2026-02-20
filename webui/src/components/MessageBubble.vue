<template>
	<div class="d-flex mb-2" :class="isMine ? 'justify-content-end' : 'justify-content-start'">
		<div
			class="p-2 rounded shadow-sm"
			:class="isMine ? 'bg-success text-white' : 'bg-white'"
			style="max-width: 60%; min-width: 120px;"
		>
			<div v-if="!isMine" class="fw-bold small text-primary mb-1">{{ message.senderName }}</div>
			<div v-if="message.content">{{ message.content }}</div>
			<div v-if="message.hasPhoto" class="text-muted">📷 Photo</div>
			<div class="d-flex justify-content-between align-items-center mt-1">
				<small :class="isMine ? 'text-white-50' : 'text-muted'">
					{{ formatTime(message.timestamp) }}
					<span v-if="isMine">{{ message.status === 'read' ? '✓✓' : '✓' }}</span>
				</small>
				<div>
					<button v-if="isMine" @click="$emit('delete')" class="btn btn-link btn-sm p-0 ms-2" title="Delete">🗑️</button>
					<button @click="$emit('react')" class="btn btn-link btn-sm p-0 ms-1" title="React">👍</button>
				</div>
			</div>
			<!-- Comments/Reactions -->
			<div v-if="message.comments && message.comments.length > 0" class="mt-1 pt-1 border-top">
				<span v-for="comment in message.comments" :key="comment.userId" class="badge bg-light text-dark me-1" :title="comment.userName">
					{{ comment.emoticon }}
				</span>
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: 'MessageBubble',
	props: {
		message: {type: Object, required: true},
		isMine: {type: Boolean, default: false},
	},
	emits: ['delete', 'react'],
	methods: {
		formatTime(timestamp) {
			if (!timestamp) return '';
			const date = new Date(timestamp);
			return date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'});
		},
	},
};
</script>
