import type { User } from "~/types";

const customers: User[] = [
  {
    id: 1,
    name: "艾力克斯·史密斯",
    email: "alex.smith@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=1",
    },
    status: "subscribed",
    location: "中国，北京",
  },
  {
    id: 2,
    name: "乔丹·布朗",
    email: "jordan.brown@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=2",
    },
    status: "unsubscribed",
    location: "中国，上海",
  },
  {
    id: 3,
    name: "泰勒·格林",
    email: "taylor.green@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=3",
    },
    status: "bounced",
    location: "中国，广州",
  },
  {
    id: 4,
    name: "摩根·怀特",
    email: "morgan.white@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=4",
    },
    status: "subscribed",
    location: "中国，深圳",
  },
  {
    id: 5,
    name: "凯西·格雷",
    email: "casey.gray@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=5",
    },
    status: "subscribed",
    location: "中国，杭州",
  },
  {
    id: 6,
    name: "杰米·约翰逊",
    email: "jamie.johnson@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=6",
    },
    status: "subscribed",
    location: "中国，成都",
  },
  {
    id: 7,
    name: "莱利·戴维斯",
    email: "riley.davis@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=7",
    },
    status: "subscribed",
    location: "中国，北京",
  },
  {
    id: 8,
    name: "凯莉·威尔逊",
    email: "kelly.wilson@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=8",
    },
    status: "subscribed",
    location: "中国，上海",
  },
  {
    id: 9,
    name: "德鲁·摩尔",
    email: "drew.moore@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=9",
    },
    status: "bounced",
    location: "中国，广州",
  },
  {
    id: 10,
    name: "乔丹·泰勒",
    email: "jordan.taylor@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=10",
    },
    status: "subscribed",
    location: "中国，深圳",
  },
  {
    id: 11,
    name: "摩根·安德森",
    email: "morgan.anderson@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=11",
    },
    status: "subscribed",
    location: "中国，杭州",
  },
  {
    id: 12,
    name: "凯西·托马斯",
    email: "casey.thomas@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=12",
    },
    status: "unsubscribed",
    location: "中国，成都",
  },
  {
    id: 13,
    name: "杰米·杰克逊",
    email: "jamie.jackson@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=13",
    },
    status: "unsubscribed",
    location: "中国，北京",
  },
  {
    id: 14,
    name: "莱利·怀特",
    email: "riley.white@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=14",
    },
    status: "unsubscribed",
    location: "中国，上海",
  },
  {
    id: 15,
    name: "凯莉·哈里斯",
    email: "kelly.harris@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=15",
    },
    status: "subscribed",
    location: "中国，广州",
  },
  {
    id: 16,
    name: "德鲁·马丁",
    email: "drew.martin@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=16",
    },
    status: "subscribed",
    location: "中国，深圳",
  },
  {
    id: 17,
    name: "艾力克斯·汤普森",
    email: "alex.thompson@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=17",
    },
    status: "unsubscribed",
    location: "中国，杭州",
  },
  {
    id: 18,
    name: "乔丹·加西亚",
    email: "jordan.garcia@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=18",
    },
    status: "subscribed",
    location: "中国，成都",
  },
  {
    id: 19,
    name: "泰勒·罗德里格斯",
    email: "taylor.rodriguez@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=19",
    },
    status: "bounced",
    location: "中国，北京",
  },
  {
    id: 20,
    name: "摩根·洛佩兹",
    email: "morgan.lopez@example.com",
    avatar: {
      src: "https://i.pravatar.cc/128?u=20",
    },
    status: "subscribed",
    location: "中国，上海",
  },
];

import { consola } from "consola";

export default eventHandler(async () => {
  consola.info("[GET /api/customers]");
  return customers;
});
