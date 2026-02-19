export interface BaseCanvasItem {
    id: string;
    name: string;
    props: Record<string, any>;
    content?: string | AnyCanvasItem[];
    locked?: boolean;
}

export interface ElementCanvasItem extends BaseCanvasItem {
    type: 'ELEMENT';
    tag: string;
    content?: string | AnyCanvasItem[];
}

export interface ComponentCanvasItem extends BaseCanvasItem {
    type: 'COMPONENT';
    componentType: string;
}

export interface IconCanvasItem extends BaseCanvasItem {
    type: 'ICON';
    iconName: string;
}

export type AnyCanvasItem = ElementCanvasItem | ComponentCanvasItem | IconCanvasItem;
