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
        Container,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Row,
    } from "sveltestrap";
    import { fly } from "svelte/transition";

    import type { GardenResponse, getGarden } from "../../lib/gardenClient";
    import { lightAction, stopAction } from "../../lib/gardenClient";

    export let garden: GardenResponse;
    export let withLink = false;

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
            <a href="#/gardens/{garden.id}">
                <CardHeader>
                    <CardTitle>
                        {garden.name}
                    </CardTitle>
                </CardHeader>
            </a>
        {:else}
            <CardHeader>
                <CardTitle>{garden.name}</CardTitle>
            </CardHeader>
        {/if}
        <CardBody>
            <CardText>
                <Container>
                    <Card>
                        <CardBody>
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
                        </CardBody>
                    </Card>
                    <Card>
                        <CardBody>
                            <Row>
                                <Col>{garden.num_zones} Zones <Icon name="grid" /></Col>
                                <Col>{garden.num_plants} Plants <Icon name="" /></Col>
                            </Row>
                        </CardBody>
                    </Card>
                    <Card>
                        <CardBody>
                            <Row>
                                {#if garden.health != null}
                                    <Col>Health Status: {garden.health.status}</Col>
                                    <Col>Health Details: {garden.health.details}</Col>
                                {:else if garden.end_date == null}
                                    <Col>No health details available</Col>
                                {/if}

                                {#if garden.temperature_humidity_data != null}
                                    <Col>
                                        Temperature: {garden.temperature_humidity_data.temperature_celsius * 1.8 + 32}°F
                                    </Col>
                                    <Col>
                                        Humidity: {garden.temperature_humidity_data.humidity_percentage}%
                                    </Col>
                                {/if}
                            </Row>
                        </CardBody>
                    </Card>
                    {#if garden.light_schedule != null}
                        <Card>
                            <CardBody>
                                <Row>
                                    <Col>
                                        Light Schedule Duration: {garden.light_schedule.duration}
                                        <Icon name="hourglass-split" />
                                    </Col>

                                    <Col>
                                        Light Schedule Start: {garden.light_schedule.start_time}
                                        <Icon name="clock" />
                                    </Col>
                                </Row>
                            </CardBody>
                        </Card>
                    {/if}
                    {#if garden.next_light_action != null}
                        <Card>
                            <CardBody>
                                <Row>
                                    <Col>
                                        Next Light Time: {garden.next_light_action.time}
                                        <Icon name="clock" />
                                    </Col>
                                    <Col>
                                        Next Light State: {garden.next_light_action.state}
                                        <Icon
                                            name={garden.next_light_action.state == "ON" ? "sunrise" : "sunset"}
                                            style="color: {garden.next_light_action.state == 'ON' ? 'orange' : 'gray'}"
                                        />
                                    </Col>
                                </Row>
                            </CardBody>
                        </Card>
                    {/if}
                </Container>
            </CardText>
        </CardBody>
        <CardFooter>
            {#if garden.end_date != null}
                <Badge color={"danger"}>End Dated</Badge>
            {/if}

            {#if garden.health != null}
                <Icon name="wifi" style="color: {garden.health.status == 'UP' ? 'green' : 'red'}" />
            {/if}

            {#if garden.next_light_action != null}
                <Icon
                    name={garden.next_light_action.state == "ON" ? "sunrise" : "sunset"}
                    style="color: {garden.next_light_action.state == 'ON' ? 'orange' : 'gray'}"
                />
            {/if}

            <Icon name="grid" />{garden.num_zones}

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
