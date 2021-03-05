-- Example query.

WITH example_data as (
    SELECT "a" as label, 111 as value
    UNION ALL
    SELECT "b" as label, 222 as value
    UNION ALL
    SELECT "b" as label, 111 as value
)

SELECT
   label, SUM(value) as value
FROM
   example_data
GROUP BY
   label
