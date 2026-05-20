let charts = {};
let currentFrom = null;
let currentTo = null;

function createChart(id, color, fixedMax) {
  const ctx = document.getElementById(id);
  const opts = {
    type: 'line',
    data: { datasets: [{ data: [], borderColor: color, borderWidth: 1.5, fill: false, pointRadius: 0, tension: 0.1 }] },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      plugins: { legend: { display: false } },
      scales: {
        x: {
          type: 'category',
          ticks: { color: '#70708a', maxTicksLimit: 10, autoSkip: true, font: { size: 10 } },
          grid: { color: '#2a2a3e' }
        },
        y: {
          beginAtZero: true,
          ticks: { color: '#70708a', font: { size: 10 } },
          grid: { color: '#2a2a3e' }
        }
      },
      elements: { point: { radius: 0 } }
    }
  };
  if (fixedMax) opts.options.scales.y.max = fixedMax;
  return new Chart(ctx, opts);
}

function initCharts() {
  charts.cpu = createChart('cpuChart', '#4fc3f7', 100);
  charts.mem = createChart('memChart', '#81c784', 100);
  charts.fpm = createChart('fpmChart', '#ffb74d');
  charts.fpmErr = createChart('fpmErrChart', '#ce93d8');
}

function fmtLabels(data) {
  if (!data.length) return [];
  const range = data[data.length - 1].created_at - data[0].created_at;
  if (range > 86400 * 10) {
    return data.map(d => {
      const dt = new Date(d.created_at * 1000);
      return dt.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    });
  }
  if (range > 86400) {
    return data.map(d => {
      const dt = new Date(d.created_at * 1000);
      return dt.toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit' });
    });
  }
  return data.map(d => {
    const dt = new Date(d.created_at * 1000);
    return dt.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
  });
}

async function fetchLatest() {
  try {
    const res = await fetch('api/latest');
    const d = await res.json();
    if (d.cpu_percent !== undefined) {
      document.getElementById('cpuValue').textContent = d.cpu_percent.toFixed(1) + '%';
      document.getElementById('memValue').textContent = d.memory_percent.toFixed(1) + '%';
      document.getElementById('fpmValue').textContent = d.fpm_active;
      document.getElementById('fpmErrValue').textContent = d.fpm_status_errors;
      const dt = new Date(d.created_at * 1000);
      document.getElementById('updatedAt').textContent = dt.toLocaleString();
    }
  } catch (_) {}
}

async function fetchMetrics(from, to) {
  try {
    const p = new URLSearchParams();
    if (from != null) p.set('from', from);
    if (to != null) p.set('to', to);
    const res = await fetch('api/metrics?' + p.toString());
    return await res.json();
  } catch (_) { return []; }
}

async function updateCharts() {
  const data = await fetchMetrics(currentFrom, currentTo);
  const labels = fmtLabels(data);

  charts.cpu.data.labels = labels;
  charts.cpu.data.datasets[0].data = data.map(d => d.cpu_percent);
  charts.cpu.update('none');

  charts.mem.data.labels = labels;
  charts.mem.data.datasets[0].data = data.map(d => d.memory_percent);
  charts.mem.update('none');

  charts.fpm.data.labels = labels;
  charts.fpm.data.datasets[0].data = data.map(d => d.fpm_active);
  charts.fpm.update('none');

  charts.fpmErr.data.labels = labels;
  charts.fpmErr.data.datasets[0].data = data.map(d => d.fpm_status_errors);
  charts.fpmErr.update('none');
}

function setPeriod(hours) {
  const now = Math.floor(Date.now() / 1000);
  currentTo = now;
  currentFrom = now - hours * 3600;
  updateCharts();
  document.querySelectorAll('.period-btn').forEach(b => b.classList.remove('active'));
  const btn = document.querySelector(`.period-btn[data-hours="${hours}"]`);
  if (btn) btn.classList.add('active');
}

document.addEventListener('DOMContentLoaded', () => {
  initCharts();
  fetchLatest();
  setPeriod(24);
  setInterval(fetchLatest, 10000);
  setInterval(() => { fetchLatest(); updateCharts(); }, 60000);
});
