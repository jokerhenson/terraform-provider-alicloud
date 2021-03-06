output "cluster_name" {
  value       = alicloud_edas_cluster.default.cluster_name
  description = "The name of the cluster that you want to create."
}

output "cluster_type" {
  value       = alicloud_edas_cluster.default.cluster_type
  description = "The type of the cluster that you want to create. Valid values: 1: Swarm cluster. 2: ECS cluster. 3: Kubernates cluster."
}

output "network_mode" {
  value       = alicloud_edas_cluster.default.network_mode
  description = "The network type of the cluster that you want to create. Valid values: 1: classic network. 2: VPC."
}

output "logical_region_id" {
  value       = alicloud_edas_cluster.default.logical_region_id
  description = "The ID of the namespace where you want to create the application."
}

output "vpc_id" {
  value       = alicloud_edas_cluster.default.vpc_id
  description = "The ID of the Virtual Private Cloud (VPC) for the cluster that you want to create. This parameter needs to be specified if the ClusterType is set as VPC."
}