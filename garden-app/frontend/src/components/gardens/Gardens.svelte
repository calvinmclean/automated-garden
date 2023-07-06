<script lang="ts">
    import { Accordion, AccordionItem } from "sveltestrap";

    import GardenCard from "./GardenCard.svelte";
    import type { GardenResponse } from "../../lib/gardenClient";

    export let gardens: GardenResponse[];

    const filterGardens = (gardens, endDated) =>
        gardens.filter((g) =>
            endDated ? g.end_date != null : g.end_date == null
        );
</script>

{#if gardens}
    {#each filterGardens(gardens, false) as garden}
        <GardenCard {garden} withLink={true} />
    {/each}

    {#if filterGardens(gardens, true).length != 0}
        <Accordion flush>
            <AccordionItem header="End Dated Gardens">
                {#each filterGardens(gardens, true) as garden}
                    <GardenCard {garden} withLink={true} />
                {/each}
            </AccordionItem>
        </Accordion>
    {/if}
{/if}
