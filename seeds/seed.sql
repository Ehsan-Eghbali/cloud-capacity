-- Demo data. Safe to run repeatedly.

INSERT INTO regions (id, name, location, status, metadata) VALUES
('ir-thr-fr1', 'Tehran - Forough Data Center', 'Tehran, IR', 'ONLINE', '{"network_tier": 3, "az": "thr-1"}'::jsonb),
('ir-tbz-1', 'Tabriz - Shahryar Data Center', 'Tabriz, IR', 'ONLINE', '{"network_tier": 2, "az": "tbz-1"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

INSERT INTO resource_types (id, name, unit, is_divisible, min_amount) VALUES
('cpu_cores', 'Virtual CPU Cores', 'cores', true, 1),
('ram_mb', 'System Memory', 'mb', true, 512),
('disk_gb', 'Block Storage', 'gb', true, 10),
('public_ip', 'Public IPv4 Address', 'count', false, 1),
('gpu_t4', 'NVIDIA Tesla T4 GPU', 'count', false, 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO region_capacities (region_id, resource_type_id, total_capacity, allocated_capacity, reserved_capacity) VALUES
('ir-thr-fr1', 'cpu_cores', 1000, 200, 0),
('ir-thr-fr1', 'ram_mb', 2048000, 409600, 0),
('ir-thr-fr1', 'disk_gb', 50000, 10000, 0),
('ir-thr-fr1', 'public_ip', 256, 50, 0),
('ir-thr-fr1', 'gpu_t4', 8, 2, 0),
('ir-tbz-1', 'cpu_cores', 400, 50, 0),
('ir-tbz-1', 'ram_mb', 819200, 102400, 0),
('ir-tbz-1', 'disk_gb', 20000, 2000, 0),
('ir-tbz-1', 'public_ip', 128, 20, 0),
('ir-tbz-1', 'gpu_t4', 2, 0, 0)
ON CONFLICT (region_id, resource_type_id) DO NOTHING;

INSERT INTO customers (id, name, email, tier, status, metadata) VALUES
('c1000000-0000-0000-0000-000000000001', 'Alborz Commerce Co.', 'infra@alborz-commerce.example', 'ENTERPRISE', 'ACTIVE', '{"contract_id": "CNT-2026-ALB"}'::jsonb),
('c2000000-0000-0000-0000-000000000002', 'Nobahar Startup', 'dev@nobahar-startup.example', 'COMMUNITY', 'ACTIVE', '{"contract_id": "CNT-2026-NBH"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

INSERT INTO customer_limits (customer_id, resource_type_id, max_total_limit, min_per_request, max_per_request, is_enabled) VALUES
('c1000000-0000-0000-0000-000000000001', 'cpu_cores', 128, 1, 32, true),
('c1000000-0000-0000-0000-000000000001', 'ram_mb', 262144, 512, 65536, true),
('c1000000-0000-0000-0000-000000000001', 'disk_gb', 5000, 10, 1000, true),
('c1000000-0000-0000-0000-000000000001', 'public_ip', 16, 1, 4, true),
('c1000000-0000-0000-0000-000000000001', 'gpu_t4', 4, 1, 2, true)
ON CONFLICT (customer_id, resource_type_id) DO NOTHING;

INSERT INTO customer_limits (customer_id, resource_type_id, max_total_limit, min_per_request, max_per_request, is_enabled) VALUES
('c2000000-0000-0000-0000-000000000002', 'cpu_cores', 16, 1, 8, true),
('c2000000-0000-0000-0000-000000000002', 'ram_mb', 32768, 512, 16384, true),
('c2000000-0000-0000-0000-000000000002', 'disk_gb', 500, 10, 200, true),
('c2000000-0000-0000-0000-000000000002', 'public_ip', 2, 1, 1, true),
('c2000000-0000-0000-0000-000000000002', 'gpu_t4', 0, 1, 0, false)
ON CONFLICT (customer_id, resource_type_id) DO NOTHING;
