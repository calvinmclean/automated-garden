<script lang="ts">
    import GardenLink from "./GardenLink.svelte";
    import type { components } from "../types/garden-app-openapi";

    type GardenResponse = components["schemas"]["GardenResponse"];

    export let gardens: GardenResponse[];
</script>

{#if gardens}
    {#each gardens as garden}
        <GardenLink {garden} />
    {/each}
{/if}
