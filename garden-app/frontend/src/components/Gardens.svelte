<script>
    import GardenLink from "./GardenLink.svelte";

    export let gardens;
</script>

{#if gardens}
    {#each gardens as garden}
        <ul>
            <li>
                <GardenLink {garden} />
            </li>
        </ul>
    {/each}
{/if}
