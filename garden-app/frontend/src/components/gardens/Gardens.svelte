<script lang="ts">
    import GardenLink from "./GardenLink.svelte";
    import type { GardenResponse } from "../../lib/gardenClient";

    export let gardens: GardenResponse[];
</script>

{#if gardens}
    {#each gardens as garden}
        <GardenLink {garden} />
    {/each}
{/if}
