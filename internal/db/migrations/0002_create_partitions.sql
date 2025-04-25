DO $$
BEGIN
    FOR i IN 0 ..30 LOOP
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS metrics_y%I_m%I_d%I
             PARTITION OF metrics
             FOR VALUES FROM (%L) TO (%L)',
            EXTRACT(YEAR FROM CURRENT_DATE),
            EXTRACT(MONTH FROM CURRENT_DATE),
            i,
            CURRENT_DATE + i * INTERVAL '1 day',
            CURRENT_DATE + (i + 1) * INTERVAL '1 day' 
        );
    END LOOP;
END $$;
