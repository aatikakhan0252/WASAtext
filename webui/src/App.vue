<template>
	<div id="wasatext-app">
		<router-view/>
	</div>
</template>

<script>
export default {
	name: 'App',
};
</script>

<style>
html, body {
	margin: 0;
	padding: 0;
	height: 100%;
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	background-color: #e5ddd5;
}

#wasatext-app {
	height: 100vh;
}
</style>
