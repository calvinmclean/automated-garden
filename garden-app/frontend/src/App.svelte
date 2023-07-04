<script>
  import Router from "svelte-spa-router";
  import routes from "./routes";
</script>

<svelte:head>
  <title>Garden App</title>

  <style>
    @import "https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css";
    @import "https://cdn.jsdelivr.net/npm/bootstrap-icons@1.9.1/font/bootstrap-icons.css";
  </style>
</svelte:head>

<h1>Garden App</h1>

<Router {routes} />
