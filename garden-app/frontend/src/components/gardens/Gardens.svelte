<script lang="ts">
    import { Accordion, AccordionItem, Container, Col, Row } from "sveltestrap";
    import GardenCard from "./GardenCard.svelte";
    import type { GardenResponse } from "../../lib/gardenClient";
    import { gardenStore } from "../../store";

    let gardens: GardenResponse[];

    const filterGardens = (gardens: GardenResponse[], endDated: boolean) =>
        gardens.filter((g) => (endDated ? g.end_date != null : g.end_date == null)).sort((a, b) => a.name.localeCompare(b.name));

    gardenStore.subscribe((value) => {
        gardens = value.gardens;
    });
</script>

{#if gardens}
    <Container>
        <Row>
            {#each filterGardens(gardens, false) as garden (garden.id)}
                <Col lg="6">
                    <GardenCard {garden} withLink={true} />
                </Col>
            {/each}
        </Row>
    </Container>

    {#if filterGardens(gardens, true).length != 0}
        <Accordion flush>
            <AccordionItem header="End Dated Gardens">
                <Container>
                    <Row>
                        {#each filterGardens(gardens, true) as garden (garden.id)}
                            <Col lg="6">
                                <GardenCard {garden} withLink={true} />
                            </Col>
                        {/each}
                    </Row>
                </Container>
            </AccordionItem>
        </Accordion>
    {/if}
{/if}
