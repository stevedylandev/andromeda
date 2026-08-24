// Dependency-free chart rendering for the habit detail page. Reads JSON
// points off #habit-chart's data attributes and draws an inline SVG: a line
// chart for numeric (int/float) habits, a day-by-day streak heatmap for bool
// habits.
(function () {
  var SVG_NS = "http://www.w3.org/2000/svg";

  function svgEl(tag, attrs) {
    var el = document.createElementNS(SVG_NS, tag);
    for (var k in attrs) el.setAttribute(k, attrs[k]);
    return el;
  }

  function pad2(n) {
    return n < 10 ? "0" + n : "" + n;
  }

  function dayKey(date) {
    return date.getFullYear() + "-" + pad2(date.getMonth() + 1) + "-" + pad2(date.getDate());
  }

  function renderLine(el, points, unit) {
    var width = Math.max(el.clientWidth || 0, 280);
    var height = 180;
    var padding = { top: 16, right: 16, bottom: 24, left: 44 };

    var vals = points.map(function (p) {
      return p.v;
    });
    var min = Math.min.apply(null, vals);
    var max = Math.max.apply(null, vals);
    if (min === max) {
      min -= 1;
      max += 1;
    }
    var pad = (max - min) * 0.1;
    min -= pad;
    max += pad;

    var t0 = points[0].t;
    var t1 = points[points.length - 1].t;
    if (t0 === t1) t1 = t0 + 1;

    var innerW = width - padding.left - padding.right;
    var innerH = height - padding.top - padding.bottom;

    function x(t) {
      return padding.left + ((t - t0) / (t1 - t0)) * innerW;
    }
    function y(v) {
      return padding.top + innerH - ((v - min) / (max - min)) * innerH;
    }

    var svg = svgEl("svg", {
      width: width,
      height: height,
      viewBox: "0 0 " + width + " " + height,
      class: "chart-svg",
    });

    [min + pad, (min + max) / 2, max - pad].forEach(function (v) {
      var ly = y(v).toFixed(1);
      svg.appendChild(
        svgEl("line", {
          x1: padding.left,
          x2: width - padding.right,
          y1: ly,
          y2: ly,
          class: "chart-gridline",
        }),
      );
      var label = svgEl("text", {
        x: padding.left - 6,
        y: ly,
        class: "chart-axis-label",
        "text-anchor": "end",
        "dominant-baseline": "middle",
      });
      label.textContent = Number(v.toFixed(2));
      svg.appendChild(label);
    });

    var d = points
      .map(function (p, i) {
        return (i === 0 ? "M" : "L") + x(p.t).toFixed(1) + "," + y(p.v).toFixed(1);
      })
      .join(" ");
    svg.appendChild(svgEl("path", { d: d, class: "chart-line" }));

    points.forEach(function (p) {
      var cx = x(p.t).toFixed(1);
      var cy = y(p.v).toFixed(1);
      var dot = svgEl("circle", { cx: cx, cy: cy, r: 3, class: "chart-dot" });
      var title = svgEl("title", {});
      title.textContent = new Date(p.t * 1000).toLocaleString() + ": " + p.v + (unit ? " " + unit : "");
      dot.appendChild(title);
      svg.appendChild(dot);
    });

    el.appendChild(svg);
  }

  function renderHeatmap(el, points) {
    var days = {};
    points.forEach(function (p) {
      var key = dayKey(new Date(p.t * 1000));
      var truthy = p.v === 1;
      if (!days[key] || truthy) days[key] = truthy ? 2 : 1;
    });

    var keys = Object.keys(days).sort();
    var start = new Date(keys[0] + "T00:00:00");
    start.setDate(start.getDate() - start.getDay());
    var today = new Date();
    today.setHours(0, 0, 0, 0);

    var totalDays = Math.round((today - start) / 86400000) + 1;
    var weeks = Math.ceil(totalDays / 7);
    var cell = 11;
    var gap = 3;
    var step = cell + gap;
    var width = weeks * step;
    var height = 7 * step;

    var svg = svgEl("svg", {
      width: width,
      height: height,
      viewBox: "0 0 " + width + " " + height,
      class: "chart-svg",
    });

    for (var i = 0; i < totalDays; i++) {
      var d = new Date(start);
      d.setDate(d.getDate() + i);
      var key = dayKey(d);
      var status = days[key] || 0;
      var week = Math.floor(i / 7);
      var dow = i % 7;

      var rect = svgEl("rect", {
        x: week * step,
        y: dow * step,
        width: cell,
        height: cell,
        rx: 2,
        class: "chart-cell chart-cell-" + status,
      });
      var title = svgEl("title", {});
      title.textContent = key + ": " + (status === 2 ? "true" : status === 1 ? "false" : "no record");
      rect.appendChild(title);
      svg.appendChild(rect);
    }

    el.appendChild(svg);
  }

  function init() {
    var el = document.getElementById("habit-chart");
    if (!el) return;

    var points;
    try {
      points = JSON.parse(el.dataset.points || "[]");
    } catch (e) {
      return;
    }
    if (!points.length) return;

    if (el.dataset.type === "bool") {
      renderHeatmap(el, points);
    } else {
      renderLine(el, points, el.dataset.unit || "");
    }
  }

  init();
})();
