<script lang="ts">
  import { Alert, Styles } from "sveltestrap";
  import Router from "svelte-spa-router";
  import routes from "./routes";
  import NavBar from "./components/NavBar.svelte";
  import { gardenStore, waterScheduleStore } from "./store";

  let theme: "dark" | "light" | "auto" = "auto";

  let demoMode = process.env.NODE_ENV == "demo";

  gardenStore.init();
  waterScheduleStore.init();
</script>

<svelte:head>
  <title>Garden App</title>

  <style>
    @import "https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css";
    @import "https://cdn.jsdelivr.net/npm/bootstrap-icons@1.9.1/font/bootstrap-icons.css";
  </style>
</svelte:head>

<Styles {theme} />

<NavBar />

{#if demoMode}
  <Alert color="warning" heading="Not all features are available in this demo" dismissible>
    This demo is a work-in-progress and I hope to add mocks for all read-only features here soon
  </Alert>
{/if}

<Router {routes} />
