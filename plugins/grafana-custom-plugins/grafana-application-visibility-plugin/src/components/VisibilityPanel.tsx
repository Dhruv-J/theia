import React from 'react';
import mermaid from 'mermaid';
import { PanelProps } from '@grafana/data';
import { VisibilityOptions } from 'types';
import { useTheme2 } from '@grafana/ui';

interface Props extends PanelProps<VisibilityOptions> {}

class Mermaid extends React.Component<any> {
  componentDidMount() {
    mermaid.contentLoaded();
  }
  render() {
    return <div className="mermaid">{this.props.chart}</div>
  }
}

export const VisibilityPanel: React.FC<Props> = ({ options, data, width, height }) => {
  const theme = useTheme2();
  const frame = data.series[0];
  const sourceIPs = frame.fields.find((field) => field.name === 'sourceIP');
  const destinationIPs = frame.fields.find((field) => field.name === 'destinationIP');
  const sourceTransportPorts = frame.fields.find((field) => field.name === 'sourceTransportPort');
  const destinationTransportPorts = frame.fields.find((field) => field.name === 'destinationTransportPort');
  const httpValsSet = frame.fields.find((field) => field.name === 'httpVals');

  let graphString = 'graph LR;\n';
  let styleString = '';

  mermaid.initialize({
    startOnLoad: true,
    theme: 'base',
    themeVariables: {
      secondaryColor: theme.colors.background.canvas,
      tertiaryColor: theme.colors.background.canvas,
      primaryTextColor: theme.colors.text.maxContrast,
      lineColor: theme.colors.text.maxContrast,
    },
  });

  function getColorFromStatus(httpStatus: string) {
    // colors that correspond to each of the types of http response code; i.e. 4xx and 5xx codes return red to symbolize errors
    const colors = ['orange', 'green', 'blue', 'red', 'red'];
    let statusType = +httpStatus.charAt(0);
    if (statusType < 1 || statusType > 5) {
      // nothing else returns purple, purple indicates an error in the httpVals field
      return 'purple';
    }
    return colors[statusType-1];
  }

  for (let i = 0; i < frame.length; i++) {
    const sourceIP = sourceIPs?.values.get(i);
    const sourcePort = sourceTransportPorts?.values.get(i);
    const destinationIP = destinationIPs?.values.get(i);
    const destinationPort = destinationTransportPorts?.values.get(i);
    const httpVals = httpValsSet?.values.get(i);
    // 0 - hostname, 1 - URL, 2 - UserAgent, 3 - ContentType, 4 - Method, 5 - Protocol, 6 - Status, 7 - ContentLength
    let vals = httpVals.split('<>');
    if (vals.length !== 8) {
      continue;
    }
    let graphLine = sourceIP + ':' + sourcePort + ' --' + vals[7] + '--> ' + destinationIP + ':' + destinationPort + '\n';
    graphString = graphString + graphLine;
    let styleLine = 'linkStyle ' + i + ' stroke: ' + getColorFromStatus(vals[6]) + '\n';
    styleString = styleString + styleLine;
  }

  // adds the styling lines to the graphstring
  graphString = graphString + styleString;

  // checking if graph syntax is valid
  mermaid.parseError = function() {
    console.log('incorrect graph syntax for graph:\n'+graphString);
    return (
      <div><p>Incorrect Graph Syntax!</p></div>
    );
  }

  mermaid.parse(graphString)
  let graphElement = document.getElementsByClassName("graphDiv")[0];
  // null check because the div does not exist at this point during the first run
  if (graphElement != null) {
    mermaid.mermaidAPI.render('graphDiv', graphString, graphElement);
  }

  // manually display first time, since render has no target yet
  return (
    <div className="graphDiv">
      <Mermaid chart={graphString}/>
    </div>
  );
};
