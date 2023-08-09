<script lang="ts">
    import {
        Badge,
        Button,
        ButtonDropdown,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardSubtitle,
        CardText,
        CardTitle,
        Col,
        Collapse,
        Container,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Popover,
        Row,
    } from "sveltestrap";
    import { fly } from "svelte/transition";

    import type { GardenResponse, getGarden } from "../../lib/gardenClient";
    import { lightAction, stopAction } from "../../lib/gardenClient";

    export let garden: GardenResponse;
    export let withLink = false;

    let lightScheduleCollapseIsOpen: boolean = false;

    function toggleLight(event) {
        lightAction(garden.id, "");
    }

    function lightOn(event) {
        lightAction(garden.id, "ON");
    }

    function lightOff(event) {
        lightAction(garden.id, "OFF");
    }

    function stopWatering(event) {
        stopAction(garden.id);
    }
</script>

<div in:fly={{ x: 50, duration: 500 }} out:fly={{ x: -50, duration: 250 }}>
    <Card class=".col-lg-4" style="margin: 5%">
        {#if withLink}
            <a href="#/gardens/{garden.id}" style="text-decoration:none">
                <CardHeader class="text-center">
                    <CardTitle>
                        {garden.name}
                        {#if garden.health != null}
                            <Badge color={garden.health.status == "UP" ? "primary" : "danger"} id={`status-badge-${garden.id}`}>
                                <Icon name={garden.health.status == "UP" ? "wifi" : "wifi-off"} />
                                {garden.health.status}
                            </Badge>
                            <Popover trigger="hover" target={`status-badge-${garden.id}`} placement="bottom" title="Health Details">
                                {garden.health.details}
                            </Popover>
                        {/if}
                    </CardTitle>
                </CardHeader>
            </a>
        {:else}
            <CardHeader>
                <CardTitle>
                    {garden.name}
                </CardTitle>
            </CardHeader>
        {/if}
        <CardBody>
            <CardText>
                <Container>
                    <Row>
                        <Col>
                            Topic prefix: {garden.topic_prefix}
                            <Icon name="globe2" />
                        </Col>
                        {#if garden.end_date != null}
                            <Col>
                                End Dated: {garden.end_date}
                                <Icon name="clock-fill" style="color: red" />
                            </Col>
                        {/if}
                    </Row>

                    <Row>
                        <Col>{garden.num_zones} Zones <Icon name="grid" /></Col>
                        <Col>{garden.num_plants} Plants <Icon name="" /></Col>
                    </Row>

                    <Row>
                        {#if garden.temperature_humidity_data != null}
                            <Col>
                                Temperature: {(garden.temperature_humidity_data.temperature_celsius * 1.8 + 32).toFixed(2)}°F
                            </Col>
                            <Col>
                                Humidity: {garden.temperature_humidity_data.humidity_percentage.toFixed(2)}%
                            </Col>
                        {/if}
                    </Row>
                    {#if garden.next_light_action != null}
                        <Card on:click={() => (lightScheduleCollapseIsOpen = !lightScheduleCollapseIsOpen)}>
                            <CardBody>
                                Light will turn {garden.next_light_action.state} at {garden.next_light_action.time}
                                <Icon
                                    name={garden.next_light_action.state == "ON" ? "sunrise" : "sunset"}
                                    style="color: {garden.next_light_action.state == 'ON' ? 'orange' : 'gray'}"
                                />
                            </CardBody>
                        </Card>

                        {#if garden.light_schedule != null}
                            <Collapse isOpen={lightScheduleCollapseIsOpen}>
                                <Card body>
                                    <Row>
                                        <Col>
                                            Duration: {garden.light_schedule.duration}
                                            <Icon name="hourglass-split" />
                                        </Col>

                                        <Col>
                                            Starting At: {garden.light_schedule.start_time}
                                            <Icon name="clock" />
                                        </Col>
                                    </Row>
                                </Card>
                            </Collapse>
                        {/if}
                    {/if}
                </Container>
            </CardText>
        </CardBody>
        <CardFooter>
            {#if garden.end_date != null}
                <Badge color="danger">End Dated</Badge>
            {/if}

            {#if garden.end_date == null}
                {#if garden.light_schedule != null}
                    <ButtonDropdown>
                        <DropdownToggle color="warning" caret>
                            <Icon name="sun" />
                        </DropdownToggle>
                        <DropdownMenu>
                            <DropdownItem on:click={toggleLight}>Toggle</DropdownItem>
                            <DropdownItem on:click={lightOn}>ON</DropdownItem>
                            <DropdownItem on:click={lightOff}>OFF</DropdownItem>
                        </DropdownMenu>
                    </ButtonDropdown>
                {/if}

                <Button color="danger" on:click={stopWatering}>
                    <Icon name="sign-stop" />
                </Button>
            {/if}
        </CardFooter>
    </Card>
</div>
