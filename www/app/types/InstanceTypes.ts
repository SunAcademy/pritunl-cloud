/// <reference path="../References.d.ts"/>
export const SYNC = 'instance.sync';
export const SYNC_NODE = 'instance.sync_node';
export const TRAVERSE = 'instance.traverse';
export const FILTER = 'instance.filter';
export const CHANGE = 'instance.change';

export interface Instance {
	id: string;
	organization?: string;
	zone?: string;
	node?: string;
	image?: string;
	status?: string;
	state?: string;
	vm_state?: string;
	public_ip?: string;
	public_ip6?: string;
	name?: string;
	memory?: number;
	processors?: number;
	network_roles?: string[];
	count?: number;
}

export interface Filter {
	name?: string;
}

export interface Info {
	instance?: string;
	firewall_rules?: string[];
	disks?: string[];
}

export type Instances = Instance[];
export type InstancesNode = Map<string, Instances>;

export type InstanceRo = Readonly<Instance>;
export type InstancesRo = ReadonlyArray<InstanceRo>;
export type InstancesNodeRo = Map<string, InstancesRo>;

export interface InstanceDispatch {
	type: string;
	data?: {
		id?: string;
		node?: string;
		instance?: Instance;
		instances?: Instances;
		page?: number;
		pageCount?: number;
		filter?: Filter;
		count?: number;
	};
}