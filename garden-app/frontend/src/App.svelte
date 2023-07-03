<script>
  import Router from "svelte-spa-router";
  import routes from "./routes";
</script>

<svelte:head>
  <title>Garden App</title>
</svelte:head>

<h1>Garden App</h1>

<Router {routes} />
