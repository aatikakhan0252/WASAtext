<template>
	<div class="d-flex justify-content-center align-items-center vh-100 bg-light">
		<div class="card shadow p-4" style="width: 400px;">
			<h2 class="text-center text-success mb-3">Welcome to WASAText</h2>
			<p class="text-center text-muted mb-4">Enter your username to start chatting</p>
			<div class="mb-3">
				<input
					v-model="username"
					@keyup.enter="doLogin"
					type="text"
					class="form-control form-control-lg"
					placeholder="Username (e.g. Maria)"
					:disabled="loading"
				/>
			</div>
			<button
				@click="doLogin"
				class="btn btn-success btn-lg w-100"
				:disabled="loading || !username"
			>
				<span v-if="loading" class="spinner-border spinner-border-sm me-2"></span>
				Start Messaging
			</button>
			<div v-if="errorMsg" class="alert alert-danger mt-3 mb-0">{{ errorMsg }}</div>
		</div>
	</div>
</template>

<script>
import api from '@/services/api.js';

export default {
	name: 'LoginView',
	data() {
		return {
			username: '',
			loading: false,
			errorMsg: null,
		};
	},
	methods: {
		async doLogin() {
			if (!this.username || this.username.length < 3 || this.username.length > 16) {
				this.errorMsg = 'Username must be between 3 and 16 characters';
				return;
			}
			this.loading = true;
			this.errorMsg = null;
			try {
				const data = await api.login(this.username);
				sessionStorage.setItem('userId', data.identifier);
				sessionStorage.setItem('userName', this.username);
				this.$router.push('/home');
			} catch (e) {
				this.errorMsg = e.response?.data?.message || 'Login failed. Please try again.';
			} finally {
				this.loading = false;
			}
		},
	},
};
</script>
