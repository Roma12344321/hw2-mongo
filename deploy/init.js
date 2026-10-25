// Инициализация replica set hw2rs из трёх узлов (адаптация initReplicaSet
// из комплекта lab.js под отдельный файл: приложение Go вместо mongosh-lab).
(async function main() {
  try {
    await rs.status();
  } catch (e) {
    if (e.code !== 94) throw e;
    await rs.initiate({
      _id: 'hw2rs',
      members: [
        {_id: 0, host: 'm1:27017'},
        {_id: 1, host: 'm2:27017'},
        {_id: 2, host: 'm3:27017'},
      ],
    });
  }

  for (let i = 0; i < 90; i++) {
    const s = await rs.status();
    if (s.members.length === 3 &&
        s.members.filter(m => m.stateStr === 'PRIMARY').length === 1 &&
        s.members.filter(m => m.stateStr === 'SECONDARY').length === 2) {
      printjson(s.members.map(m => ({host: m.name, role: m.stateStr})));
      return;
    }
    await sleep(1000);
  }

  throw Error('Replica set не готов: docker compose logs init m1 m2 m3');
})().catch(e => { console.error(e); quit(1); });
