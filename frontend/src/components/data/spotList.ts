export const commandMenuItems = [
    {
        group: '应用',
        items: [
            { id: 'app-01', name: 'MySQL' },
            { id: 'app-02', name: 'PostgreSQL' },
            { id: 'app-03', name: 'MongoDB' },
            { id: 'app-04', name: 'Redis' },
            { id: 'app-05', name: 'Elasticsearch' },
            { id: 'app-06', name: 'RabbitMQ' },
            { id: 'app-07', name: 'Kafka' },
            { id: 'app-08', name: 'Docker' },
            { id: 'app-09', name: 'Kubernetes' },
        ],
        icon: 'lucide:package',
    },
    {
        group: '环境',
        items: [
            { id: 'env-01', name: '开发环境' },
            { id: 'env-02', name: '测试环境' },
            { id: 'env-03', name: '生产环境' },
        ],
        icon: 'lucide:grid-2x2',
    },
    {
        group: '项目',
        items: [
            { id: 'project-01', name: '我的个人项目' },
            { id: 'project-02', name: '团队项目' },
            { id: 'project-03', name: '开源项目' },
            { id: 'project-04', name: '商业项目' },
            { id: 'project-05', name: '个人博客' },
            { id: 'project-06', name: '在线简历' },
        ],
        icon: 'lucide:folder-plus',
    },
]