<script lang="ts">
    import GardenCard from "./GardenCard.svelte";
    import type { GardenResponse } from "../../lib/gardenClient";

    export let gardens: GardenResponse[];
</script>

{#if gardens}
    {#each gardens as garden}
        <GardenCard {garden} withLink={true} />
    {/each}
{/if}
