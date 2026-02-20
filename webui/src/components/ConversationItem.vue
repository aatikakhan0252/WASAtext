<template>
	<div
		class="d-flex align-items-center p-3 border-bottom conversation-item"
		:class="{'bg-success bg-opacity-10': active}"
		@click="$emit('select')"
		role="button"
	>
		<div class="flex-grow-1 ms-2 overflow-hidden">
			<div class="d-flex justify-content-between">
				<strong class="text-truncate">{{ conversation.name || 'Unknown' }}</strong>
				<small class="text-muted ms-2 text-nowrap">{{ formatTime(conversation.lastMessageTimestamp) }}</small>
			</div>
			<div class="text-muted text-truncate small">
				{{ conversation.lastMessageIsPhoto ? '📷 Photo' : (conversation.lastMessagePreview || 'No messages yet') }}
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: 'ConversationItem',
	props: {
		conversation: {type: Object, required: true},
		active: {type: Boolean, default: false},
	},
	emits: ['select'],
	methods: {
		formatTime(timestamp) {
			if (!timestamp) return '';
			const date = new Date(timestamp);
			return date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'});
		},
	},
};
</script>

<style scoped>
.conversation-item:hover {
	background-color: #f0f2f5;
}
</style>
