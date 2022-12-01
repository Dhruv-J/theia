# Grafana Service Dependency Graph Plugin

## What is Grafana Service Dependency Graph Plugin?

Grafana Service Dependency Graph Plugin allows users to visualize how pods send
and receive traffic to and from other pods and services. To help visualize the
flows, the plugin groups pods by pod label and shows them 'inside' the node they
are a part of.

```mermaid
graph LR;
subgraph worker_node
worker_pod_1
worker_pod_2
end;
```
A line pointing from source to destination is also shown, with the amount of
data transmitted labelling the line.
```mermaid
graph LR;
subgraph worker_node
worker_pod_1 -- 8.4 MB --> worker_pod_2
end;
```
## Acknowledgements

The Service Dependency Graph Plugin is created using [mermaid-js](https://mermaid-js.github.io/mermaid/#/)

## Data Source

Supported Databases:

- ClickHouse

## Queries Convention

Currently the Service Dependency Graph Plugin is created for restricted uses,
only for visualizing transmitted data between pods and services. For correct
loading of data for the Service Dependency Graph Plugin, the query is expected
to return the following fields, in arbitrary order.

- field 1: sourcePodName value with name or an alias of `sourcePodName`
- field 2: sourcePodLabels value with name or an alias of `sourcePodLabels`
- field 3: sourceNodeName value with name or an alias of `sourceNodeName`
- field 4: destinationPodName value with name or an alias of `destinationPodName`
- field 5: destinationPodLabels value with name or an alias of `destinationPodLabels`
- field 6: destinationNodeName value with name or an alias of `destinationNodeName`
- field 7: destinationServicePortName value with name or an alias of `destinationServicePortName`
- field 8: octetDeltaCount value with name or an alias of `octetDeltaCount`

Clickhouse query example:
```sql
SELECT sourcePodName,
sourcePodLabels,
sourceNodeName,
destinationPodName,
destinationPodLabels,
destinationNodeName,
destinationServicePortName,
octetDeltaCount
FROM flows
WHERE ( destinationPodName IS NOT NULL AND destinationPodName != '' AND destinationPodName != 'undefined' )
AND ( sourcePodName IS NOT NULL AND sourcePodName != '' AND sourcePodName != 'undefined' )
AND ( positionCaseInsensitiveUTF8('${querySourcePodNamespace:raw}', sourcePodNamespace) > 0 )
AND ( positionCaseInsensitiveUTF8('${queryDestinationPodNamespace:raw}', destinationPodNamespace) > 0 )
AND ( positionCaseInsensitiveUTF8('${queryFlowType:raw}', CAST(flowType AS varchar)) > 0 )
LIMIT ${queryNumFlows}
```

## Installation

### 1. Install the Panel

Installing on a local Grafana:

For local instances, plugins are installed and updated via a simple CLI command.
Use the grafana-cli tool to install chord-panel-plugin from the commandline:

```shell
grafana-cli --pluginUrl https://downloads.antrea.io/artifacts/grafana-custom-plugins/theia-grafana-dependency-plugin-1.0.0.zip plugins install theia-grafana-dependency-plugin
```

The plugin will be installed into your grafana plugins directory; the default is
`/var/lib/grafana/plugins`. More information on the [cli tool](https://grafana.com/docs/grafana/latest/administration/cli/#plugins-commands).

Alternatively, you can manually download the .zip file and unpack it into your grafana
plugins directory.

[Download](https://downloads.antrea.io/artifacts/grafana-custom-plugins/theia-grafana-dependency-plugin-1.0.0.zip)

Installing to a Grafana deployed on Kubernetes:

In Grafana deployment manifest, configure the environment variable `GF_INSTALL_PLUGINS`
as below:

```yaml
env:
- name: GF_INSTALL_PLUGINS
   value: "https://downloads.antrea.io/artifacts/grafana-custom-plugins/theia-grafana-dependency-plugin-1.0.0.zip;theia-grafana-dependency-plugin"
```

### 2. Add the Panel to a Dashboard

Installed panels are available immediately in the Dashboards section in your Grafana
main menu, and can be added like any other core panel in Grafana. To see a list of
installed panels, click the Plugins item in the main menu. Both core panels and
installed panels will appear. For more information, visit the docs on [Grafana plugin installation](https://grafana.com/docs/grafana/latest/plugins/installation/).

## Customization

This plugin is built with [@grafana/toolkit](https://www.npmjs.com/package/@grafana/toolkit),
which is a CLI that enables efficient development of Grafana plugins. To customize
the plugin and do local testings:

1. Install dependencies

   ```bash
   cd grafana-dependency-plugin
   yarn install
   ```

2. Build plugin in development mode or run in watch mode

   ```bash
   yarn dev
   ```

   or

   ```bash
   yarn watch
   ```

3. Build plugin in production mode

   ```bash
   yarn build
   ```

## Learn more

- [Build a panel plugin tutorial](https://grafana.com/tutorials/build-a-panel-plugin)
- [Grafana documentation](https://grafana.com/docs/)
- [Grafana Tutorials](https://grafana.com/tutorials/) - Grafana Tutorials are step-by-step
guides that help you make the most of Grafana
- [Grafana UI Library](https://developers.grafana.com/ui) - UI components to help you build interfaces using Grafana Design System
