-- 任务介绍保留任务事实；来源字段只写可追溯的资料来源。
UPDATE moon_spacecraft
SET description = '运行于地月拉格朗日 L2 点附近的晕轨道，为嫦娥六号等月背采样任务提供地月中继通信，并携带极紫外相机等科学载荷。',
    source_name = 'CNSA 公开任务资料'
WHERE id = 'queqiao2';

UPDATE moon_spacecraft
SET description = '嫦娥四号月背探测任务的地月拉格朗日 L2 中继星，人类首个地月 L2 晕轨道通信卫星。',
    source_name = 'CNSA 公开任务资料'
WHERE id = 'queqiao';

UPDATE moon_spacecraft
SET source_name = 'NASA CAPSTONE 任务资料'
WHERE id = 'capstone';
