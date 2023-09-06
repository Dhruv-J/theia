import { PanelPlugin } from '@grafana/data';
import { VisibilityOptions } from './types';
import { VisibilityPanel } from './components/VisibilityPanel';

export const plugin = new PanelPlugin<VisibilityOptions>(VisibilityPanel).setPanelOptions((builder) => {
});
